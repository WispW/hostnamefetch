//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")

	procSetConsoleOutputCP       = kernel32.NewProc("SetConsoleOutputCP")
	procGetConsoleMode           = kernel32.NewProc("GetConsoleMode")
	procGetConsoleScreenBufferIn = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetStdHandle             = kernel32.NewProc("GetStdHandle")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procGetTickCount64           = kernel32.NewProc("GetTickCount64")
	procGlobalMemoryStatusEx     = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpaceExW      = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetComputerNameExW       = kernel32.NewProc("GetComputerNameExW")
	procGetActiveProcessorCount  = kernel32.NewProc("GetActiveProcessorCount")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procRtlGetVersion            = ntdll.NewProc("RtlGetVersion")

	// kernel32 handle / flag constants
	stdOutputHandle          = uintptr(^uint32(10)) // -11
	th32csSnapProcess        = uint32(0x00000002)
	allProcessorGroups       = uint32(0xFFFF)
	computerNameDNSHostname  = uint32(1)
	computerNameDNSQualified = uint32(3)

	registryMachine = syscall.Handle(syscall.HKEY_LOCAL_MACHINE)
)

type osVersionInfoW struct {
	Size       uint32
	Major      uint32
	Minor      uint32
	Build      uint32
	PlatformID uint32
	CSDVersion [128]uint16
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type consoleScreenBufferInfo struct {
	Size              [2]int16
	CursorPosition    [2]int16
	Attributes        uint16
	Window            [4]int16
	MaximumWindowSize [2]int16
}

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// collect gathers the Windows snapshot. Every value is best-effort: items that
// cannot be read stay empty and their row is skipped.
func collect() (environment, *systemInfo) {
	console := isConsole(stdoutHandle())

	env := environment{
		isTTY:     console,
		termWidth: consoleWidth(),
		noColor:   hasNoColor(),
	}

	if !console {
		// Redirected output behaves like the bash version: COLUMNS wins,
		// otherwise fall back to 80.
		env.termWidth = 80
		if value, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && value > 0 {
			env.termWidth = value
		}
	}

	switch {
	case !console:
		env.blocks = true
	default:
		env.blocks = setConsoleUTF8()
	}

	info := &systemInfo{
		User:     firstNonEmpty(os.Getenv("USER"), os.Getenv("USERNAME"), "user"),
		Hostname: computerName(computerNameDNSHostname),
	}
	if info.Hostname == "" {
		if host, err := os.Hostname(); err == nil {
			info.Hostname = host
		}
	}

	if fqdn := strings.TrimSuffix(computerName(computerNameDNSQualified), "."); strings.Contains(fqdn, ".") && fqdn != info.Hostname {
		info.FQDN = fqdn
	}

	major, minor, build := windowsVersion()
	version := fmt.Sprintf("%d.%d.%d", major, minor, build)
	arch := runtimeArch()

	info.Distro = "Windows " + version + " " + arch
	info.Kernel = version
	info.Model = machineModel()
	info.Uptime = uptime()
	info.Shell = shellName()
	info.Term = terminalName()
	info.Resolution = resolution()
	info.CPU = cpuName()
	info.GPU = gpuName()
	info.Memory = memory()
	info.LocalIP = localIP()
	info.Disk = disk()

	return env, info
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func runtimeArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "aarch64"
	case "386":
		return "i686"
	}
	return "x86_64"
}

func stdoutHandle() uintptr {
	handle, _, _ := procGetStdHandle.Call(stdOutputHandle)
	return handle
}

func isConsole(handle uintptr) bool {
	var mode uint32
	result, _, _ := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	return result != 0
}

func setConsoleUTF8() bool {
	result, _, _ := procSetConsoleOutputCP.Call(65001)
	return result != 0
}

func consoleWidth() int {
	var info consoleScreenBufferInfo
	result, _, _ := procGetConsoleScreenBufferIn.Call(
		stdoutHandle(), uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		return 80
	}

	width := int(info.Window[2]) - int(info.Window[0]) + 1
	if width < 1 {
		return 80
	}
	return width
}

func hasNoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

func windowsVersion() (uint32, uint32, uint32) {
	info := osVersionInfoW{Size: uint32(unsafe.Sizeof(osVersionInfoW{}))}
	if result, _, _ := procRtlGetVersion.Call(uintptr(unsafe.Pointer(&info))); result != 0 {
		return 0, 0, 0
	}
	return info.Major, info.Minor, info.Build
}

func computerName(nameType uint32) string {
	var buf [256]uint16
	size := uint32(len(buf))

	result, _, _ := procGetComputerNameExW.Call(
		uintptr(nameType),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)))
	if result == 0 {
		return ""
	}

	return syscall.UTF16ToString(buf[:size])
}

func resolution() string {
	width, _, _ := procGetSystemMetrics.Call(0)
	height, _, _ := procGetSystemMetrics.Call(1)
	if width == 0 || height == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", width, height)
}

func uptime() string {
	ticks, _, _ := procGetTickCount64.Call()
	seconds := uint64(ticks) / 1000

	days := seconds / 86400
	hours := seconds % 86400 / 3600
	minutes := seconds % 3600 / 60

	var parts []string
	if days > 0 {
		parts = append(parts, plural(days, "day", "days"))
	}
	if hours > 0 {
		parts = append(parts, plural(hours, "hour", "hours"))
	}
	if minutes > 0 {
		parts = append(parts, plural(minutes, "min", "mins"))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d secs", seconds)
	}
	return strings.Join(parts, ", ")
}

func plural(n uint64, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func memory() string {
	status := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	result, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if result == 0 || status.TotalPhys == 0 {
		return ""
	}

	used := status.TotalPhys - status.AvailPhys
	return fmt.Sprintf("%dMiB / %dMiB (%d%%)",
		used/1024/1024,
		status.TotalPhys/1024/1024,
		used*100/status.TotalPhys)
}

func disk() string {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = "C:"
	}
	path := drive + `\`

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	result, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)))
	if result == 0 || totalBytes == 0 {
		return ""
	}

	used := totalBytes - totalFreeBytes
	pct := used * 100 / totalBytes
	return fmt.Sprintf("%s / %s (%d%%)", humanSize(used), humanSize(totalBytes), pct)
}

// humanSize mirrors df -h: one decimal below 10 units, none above.
func humanSize(bytes uint64) string {
	units := []string{"B", "K", "M", "G", "T", "P"}
	value := float64(bytes)
	unit := 0

	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}

	if unit == 0 {
		return fmt.Sprintf("%.0f%s", value, units[unit])
	}
	if value >= 10 {
		return fmt.Sprintf("%.0f%s", value, units[unit])
	}
	return fmt.Sprintf("%.1f%s", value, units[unit])
}

func localIP() string {
	var fallback string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			if isVirtualAdapter(iface.Name) {
				if fallback == "" {
					fallback = ip4.String()
				}
				continue
			}
			return ip4.String()
		}
	}

	return fallback
}

func isVirtualAdapter(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{"vethernet", "wsl", "hyper-v", "vmware", "virtualbox", "loopback", "bluetooth"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func machineModel() string {
	const biosKey = `HARDWARE\DESCRIPTION\System\BIOS`

	manufacturer := cleanGeneric(regString(registryMachine, biosKey, "SystemManufacturer"))
	product := cleanGeneric(regString(registryMachine, biosKey, "SystemProductName"))
	board := cleanGeneric(regString(registryMachine, biosKey, "BaseBoardProduct"))

	name := strings.TrimSpace(strings.Join(nonEmpty(manufacturer, product), " "))
	if name == "" {
		return board
	}
	return name
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func cleanGeneric(value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "", "system product name", "system manufacturer", "system version",
		"to be filled by o.e.m.", "default string", "not applicable", "none":
		return ""
	}
	return trimmed
}

func cpuName() string {
	const cpuKey = `HARDWARE\DESCRIPTION\System\CentralProcessor\0`

	name := strings.Join(strings.Fields(regString(registryMachine, cpuKey, "ProcessorNameString")), " ")
	if name == "" {
		return ""
	}

	// The registry name often already contains a base clock ("... CPU @ 3.70GHz");
	// drop it so the frequency we append is not duplicated.
	if idx := strings.Index(name, "@"); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}

	cleaned := name
	replacer := strings.NewReplacer(
		"(TM)", "", "(tm)", "", "(R)", "", "(r)", "",
		"CPU", "", "Processor", "", "Dual-Core", "", "Quad-Core", "",
		"Six-Core", "", "Eight-Core", "", "Core ", "",
	)
	cleaned = strings.Join(strings.Fields(replacer.Replace(cleaned)), " ")

	cores := activeProcessorCount()
	if cores > 0 {
		cleaned += fmt.Sprintf(" (%d)", cores)
	}

	if mhz := regDWORD(registryMachine, cpuKey, "~MHz"); mhz > 0 {
		if mhz >= 1000 {
			cleaned += fmt.Sprintf(" @ %d.%dGHz", mhz/1000, mhz%1000/100)
		} else {
			cleaned += fmt.Sprintf(" @ %dMHz", mhz)
		}
	}

	return cleaned
}

func activeProcessorCount() uint32 {
	count, _, _ := procGetActiveProcessorCount.Call(uintptr(allProcessorGroups))
	return uint32(count)
}

func gpuName() string {
	const displayClass = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`

	seen := map[string]bool{}
	for _, subkey := range regSubKeys(registryMachine, displayClass) {
		desc := strings.Join(strings.Fields(regString(registryMachine, displayClass+`\`+subkey, "DriverDesc")), " ")
		if desc == "" || seen[desc] {
			continue
		}
		return desc
	}
	return ""
}

func shellName() string {
	processes := processSnapshot()
	current := uint32(os.Getpid())
	parent := processes[current].ParentProcessID

	for depth := 0; depth < 8 && parent != 0; depth++ {
		entry, ok := processes[parent]
		if !ok {
			break
		}

		switch strings.ToLower(entry.Name) {
		case "powershell.exe":
			return "Windows PowerShell"
		case "pwsh.exe":
			return "PowerShell"
		case "cmd.exe":
			return "cmd"
		case "bash.exe":
			return "bash"
		case "wsl.exe":
			return "wsl"
		}

		parent = entry.ParentProcessID
	}

	if comspec := os.Getenv("ComSpec"); comspec != "" {
		return strings.TrimSuffix(filepath.Base(comspec), filepath.Ext(comspec))
	}
	return "cmd"
}

func terminalName() string {
	if os.Getenv("WT_SESSION") != "" {
		return "Windows Terminal"
	}

	processes := processSnapshot()
	current := uint32(os.Getpid())
	parent := processes[current].ParentProcessID

	for depth := 0; depth < 8 && parent != 0; depth++ {
		entry, ok := processes[parent]
		if !ok {
			break
		}

		switch strings.ToLower(entry.Name) {
		case "windowsterminal.exe":
			return "Windows Terminal"
		case "code.exe", "code - insiders.exe":
			return "VS Code"
		case "conemu64.exe", "conemu.exe":
			return "ConEmu"
		}

		parent = entry.ParentProcessID
	}

	return "conhost"
}

type processInfo struct {
	Name            string
	ParentProcessID uint32
}

func processSnapshot() map[uint32]processInfo {
	result := map[uint32]processInfo{}

	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(th32csSnapProcess), 0)
	if snapshot == uintptr(syscall.InvalidHandle) || snapshot == 0 {
		return result
	}
	defer procCloseHandle.Call(snapshot)

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ok != 0 {
		result[entry.ProcessID] = processInfo{
			Name:            syscall.UTF16ToString(entry.ExeFile[:]),
			ParentProcessID: entry.ParentProcessID,
		}

		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		ok, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}

	return result
}

// --- registry helpers -------------------------------------------------------

func regString(root syscall.Handle, path, name string) string {
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(root, syscall.StringToUTF16Ptr(path), 0, syscall.KEY_READ, &key); err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)

	var valueType uint32
	var buf [2048]byte
	size := uint32(len(buf))

	err := syscall.RegQueryValueEx(key, syscall.StringToUTF16Ptr(name), nil, &valueType, &buf[0], &size)
	if err != nil || (valueType != syscall.REG_SZ && valueType != syscall.REG_EXPAND_SZ) {
		return ""
	}

	return decodeUTF16(buf[:size])
}

func regDWORD(root syscall.Handle, path, name string) uint32 {
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(root, syscall.StringToUTF16Ptr(path), 0, syscall.KEY_READ, &key); err != nil {
		return 0
	}
	defer syscall.RegCloseKey(key)

	var valueType uint32
	var buf [4]byte
	size := uint32(len(buf))

	if err := syscall.RegQueryValueEx(key, syscall.StringToUTF16Ptr(name), nil, &valueType, &buf[0], &size); err != nil {
		return 0
	}
	if valueType != syscall.REG_DWORD || size < 4 {
		return 0
	}

	return binary.LittleEndian.Uint32(buf[:4])
}

func regSubKeys(root syscall.Handle, path string) []string {
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(root, syscall.StringToUTF16Ptr(path), 0, syscall.KEY_READ, &key); err != nil {
		return nil
	}
	defer syscall.RegCloseKey(key)

	var out []string
	for index := uint32(0); ; index++ {
		var name [256]uint16
		size := uint32(len(name))

		err := syscall.RegEnumKeyEx(key, index, &name[0], &size, nil, nil, nil, nil)
		if err != nil {
			break
		}

		out = append(out, syscall.UTF16ToString(name[:size]))
	}

	return out
}

func decodeUTF16(data []byte) string {
	if len(data) < 2 {
		return ""
	}

	words := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		words = append(words, binary.LittleEndian.Uint16(data[i:i+2]))
	}

	for len(words) > 0 && words[len(words)-1] == 0 {
		words = words[:len(words)-1]
	}

	return string(utf16.Decode(words))
}
