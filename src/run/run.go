package run

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/yuk7/wsldl/lib/utils"
	"github.com/yuk7/wsldl/lib/wslapi"
	"github.com/yuk7/wsldl/lib/wtutils"
)

//Execute is default run entrypoint.
func Execute(name string, args []string) {
	command := ""
	for _, s := range args {
		command = command + " " + utils.DQEscapeString(s)
	}

	exitCode, err := wslapi.WslLaunchInteractive(name, command, true)
	var errno syscall.Errno
	if errors.As(err, &errno) {
		fmt.Printf("ERR: Launch Process failed\n")
		fmt.Printf("Code: 0x%x\nExit Code:0x%x", int(errno), exitCode)
		log.Fatal(err)
	} else {
		os.Exit(int(exitCode))
	}
}

//ExecuteP runs Execute function with Path Translator
func ExecuteP(name string, args []string) {
	var convArgs []string
	for _, s := range args {
		if strings.Contains(s, "\\") {
			s = strings.Replace(s, "\\", "/", -1)
			s = utils.DQEscapeString(s)
			out, exitCode, err := ExecRead(name, "wslpath -u "+s)
			if err != nil || exitCode != 0 {
				fmt.Println("ERR: Failed to Path Translation")
				var errno syscall.Errno
				if errors.As(err, &errno) {
					fmt.Printf("Code: 0x%x\n", int(errno))
				}
				fmt.Printf("ExitCode: 0x%x\n", exitCode)
				if err != nil {
					log.Fatal(err)
				}
				os.Exit(int(exitCode))
			}
			convArgs = append(convArgs, out)
		} else {
			convArgs = append(convArgs, s)
		}
	}

	Execute(name, convArgs)
}

// ExecuteNoArgs runs distro, but use terminal settings
func ExecuteNoArgs(name string) {
	efPath, _ := os.Executable()
	b, err := utils.IsParentConsole()
	if err != nil {
		b = true
	}
	if !b {
		uuid, err := utils.WslGetUUID(name)
		if err != nil {
			Execute(name, nil)
		}
		info, _ := utils.WsldlGetTerminalInfo(uuid)
		switch info {
		case utils.FlagWsldlTermWT:
			utils.FreeConsole()
			ExecWindowsTerminal(name)
			os.Exit(0)

		case utils.FlagWsldlTermFlute:
			utils.FreeConsole()
			exe := os.Getenv("LOCALAPPDATA")
			exe = utils.DQEscapeString(exe + "\\Microsoft\\WindowsApps\\flute.exe")

			cmd := exe + " run " + utils.DQEscapeString(efPath) + " run"
			res, err := utils.CreateProcessAndWait(cmd)
			if err != nil {
				utils.AllocConsole()
				println("ERR: Failed to launch Terminal Process")
				println(exe)
				println(err.Error())
				bufio.NewReader(os.Stdin).ReadString('\n')
			}
			os.Exit(res)
		}
	}
	Execute(name, nil)
}

//ExecRead execs command and read output
func ExecRead(name, command string) (out string, exitCode uint32, err error) {
	stdin := syscall.Handle(0)
	stdout := syscall.Handle(0)
	stdintmp := syscall.Handle(0)
	stdouttmp := syscall.Handle(0)
	sa := syscall.SecurityAttributes{InheritHandle: 1, SecurityDescriptor: 0}

	syscall.CreatePipe(&stdin, &stdintmp, &sa, 0)
	syscall.CreatePipe(&stdout, &stdouttmp, &sa, 0)

	handle, err := wslapi.WslLaunch(name, command, true, stdintmp, stdouttmp, stdouttmp)
	syscall.WaitForSingleObject(handle, syscall.INFINITE)
	syscall.GetExitCodeProcess(handle, &exitCode)
	buf := make([]byte, syscall.MAX_LONG_PATH)
	var length uint32

	syscall.ReadFile(stdout, buf, &length, nil)

	//[]byte -> string and cut to fit the length
	out = string(buf)[:length]
	if out[len(out)-1:] == "\n" {
		out = out[:len(out)-1]
	}
	return
}

// ExecWindowsTerminal executes Windows Terminal
func ExecWindowsTerminal(name string) {

	profileName := ""
	conf, err := wtutils.ReadParseWTConfig()
	if err == nil {
		guid := "{" + wtutils.CreateProfileGUID(name) + "}"
		for _, profile := range conf.Profiles.ProfileList {
			if profile.GUID == guid {
				profileName = profile.Name
				break
			}
		}
		if profileName == "" {
			for _, profile := range conf.Profiles.ProfileList {
				if strings.EqualFold(profile.Name, name) {
					profileName = profile.Name
					break
				}
			}
		}
	}

	exe := os.Getenv("LOCALAPPDATA")
	exe = utils.DQEscapeString(exe + "\\Microsoft\\WindowsApps\\wt.exe")
	cmd := exe

	if profileName != "" {
		cmd = cmd + " -p " + utils.DQEscapeString(profileName)
	} else {
		efPath, _ := os.Executable()
		cmd = cmd + " " + utils.DQEscapeString(efPath) + " run"
	}

	res, err := utils.CreateProcessAndWait(cmd)
	if err != nil {
		utils.AllocConsole()
		println("ERR: Failed to launch Terminal Process")
		println(exe)
		println(err.Error())
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(res)
}

// ShowHelp shows help message
func ShowHelp(showTitle bool) {
	if showTitle {
		println("Usage:")
	}
	println("    <no args>")
	println("      - Open a new shell with your default settings.")
	println()
	println("    run <command line>")
	println("      - Run the given command line in that distro. Inherit current directory.")
	println()
	println("    runp <command line (includes windows path)>")
	println("      - Run the path translated command line in that distro.")
}
