/*
 +--------------------------------------------------------------------------------+
 | 2015-05 Luca R - initially created to support a containerized Caché            |
 |                                                                                |
 |                                                                                |
 +--------------------------------------------------------------------------------+
*/

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/hpcloud/tail"
)

const (
	version         = "0.4"
	dbg             = false
	k316            = 3.16 // the kernel version that allows containers to use more useful ssmmmax seg value
	pre316MaxShmall = 8192
)

// setting shmmax
// because the linux default in kernel =<v3.16 is only 32MB and
// therefore insufficient to install and run Caché
//
// shmmaxVal param supplied in MB; however it's applied in bytes
//
func setSharedMemSeg(shmmaxVal int) (bool, error) {
	if dbg {
		log.Printf("Setting shmmax; param shmmaxVal=%d\n", shmmaxVal)
	}

	// Avoid catastrophes & having to deal with other pre-3.16 kernels settings
	// and many more extra OS / kernel param values.
	// Users should upgrade to =>3.16
	// in pre-3.16 SHMALL is 8192MB so one single segment (shmmax) cannot be larger than that
	if shmmaxVal > pre316MaxShmall {
		log.Printf("Warning: std pre-3.16 linux kernel has only 8192MB of total shared memory (shmall); Setting shmmax to this limit (8GB)")
		shmmaxVal = pre316MaxShmall
	}

	shmmaxByteVal := (shmmaxVal * 1024 * 1024)
	if dbg {
		log.Printf("shmmaxByteVal: %d", shmmaxByteVal)
	}

	// concatenating the cmd string
	kstr := []string{"kernel.shmmax=", strconv.Itoa(shmmaxByteVal)}
	kernelParamShmmax := strings.Join(kstr, "")
	if dbg {
		log.Printf("kernelParamShmmax: %s", kernelParamShmmax)
	}

	// the OS command to set the new shmmax kernel param
	cmd := "sysctl"
	args := []string{"-w", kernelParamShmmax}

	// and its execution
	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Printf("Error & possible causes:\n")
		log.Printf("-insufficient privileges to run sysctl; you must be on < 3.16 kernel\n")
		log.Printf("-container running without --privileged flag (needed for setting shared mem)\n")
		log.Printf("ERR: %s", err)
		os.Exit(1)
	}

	return true, nil
}

// returns instalation folder for given instance
func getInstanceFolder(inst string) string {
	var folder string
	cmd := "ccontrol"
	args := []string{"qlist", inst}

	// Output runs the command and returns its standard output
	if out, err := exec.Command(cmd, args...).Output(); err != nil {
		log.Printf("Error while getting Cachéinstallation folders\n")
		log.Printf("ERR: %s.", err)
		os.Exit(1)

	} else {
		// ccontrol qlist string examples:
		// C151^/usr/cachesys^2015.1.0.429.0^running, since Mon Jun  8 12:00:30 2015^cache.cpf^1972^57772^62972^warn^
		// CACHE142^/Users/CACHE142^2014.2.0.177.0^sign-on inhibited, last used Mon Jun  8 11:31:37 2015^cache.cpf^1972^57772^62972^
		// C151^/usr/cachesys^2015.1.0.429.0^down, last used Mon Jun  8 16:40:07 2015^cache.cpf^1972^57772^62972^^
		qlistStr := string(out)
		if qlistStr == "" {
			log.Printf("Error: Cannot continue as qlistStr from 'ccontrol qlist <instance>' is empty")
			os.Exit(1)
		}

		// parsing the returned string; they're all []string...
		folder = strings.SplitN(qlistStr, "^", 4)[1]
	}

	return folder
}

// shows all new lines in cconsole.log
func tailCConsoleLog(inst string) {
	folder := getInstanceFolder(inst)
	endLocation := tail.SeekInfo{Offset: 0, Whence: os.SEEK_END}
	if t, err := tail.TailFile(folder+"/mgr/cconsole.log", tail.Config{Follow: true, Location: &endLocation}); err != nil {
		log.Printf("Error while getting content for cconsole.log\n")
		log.Printf("ERR: %s.\n", err)
	} else {
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
	}
}

// starting Caché
//
func startCaché(inst string, nostu bool, cclog bool) (bool, error) {
	log.Printf("Starting Caché...\n")

	// building the start string
	cmd := "ccontrol"
	args := []string{"start"}
	args = append(args, inst)
	if nostu == true {
		args = append(args, "nostu")
	}
	args = append(args, "quietly")

	if dbg {
		log.Printf("Caché start cmd: %s %q", cmd, args)
	}

	if cclog {
		go tailCConsoleLog(inst)
	}

	c := exec.Command(cmd, args...)

	// preparing for stdout/stderr msg
	var out bytes.Buffer
	c.Stdout = &out

	// run it
	if err := c.Run(); err != nil {
		errMsg := out.String()
		log.Printf("Error & possible causes:\n")
		log.Printf("-Caché was not installed successfully")
		log.Printf("-wrong Caché instance name")
		log.Printf("-missing privileges to start/stop Caché; proc not in Caché group.")
		log.Printf("ERR: %s; %s", err, errMsg)
		os.Exit(1)
	}

	// check that the start-up was successful
	if err := checkCmdOutcome("up", inst); err != nil {
		log.Printf("Error: Caché was not brought up successfully.\n")
		os.Exit(1)
	}

	return true, nil
}

// check if cstart or cstop were successfull
// what =	what we are checking:
//			"up" checks for successful Caché start-up
// 			"down" checks for successful Caché shutdown
//
func checkCmdOutcome(what string, inst string) error {
	cmd := "ccontrol"
	args := []string{"qlist", inst}

	// Output runs the command and returns its standard output
	if out, err := exec.Command(cmd, args...).Output(); err != nil {
		log.Printf("Error while verifying Caché '%s' status\n", what)
		log.Printf("ERR: %s.", err)
		os.Exit(1)

	} else {
		// ccontrol qlist string examples:
		// C151^/usr/cachesys^2015.1.0.429.0^running, since Mon Jun  8 12:00:30 2015^cache.cpf^1972^57772^62972^warn^
		// CACHE142^/Users/CACHE142^2014.2.0.177.0^sign-on inhibited, last used Mon Jun  8 11:31:37 2015^cache.cpf^1972^57772^62972^
		// C151^/usr/cachesys^2015.1.0.429.0^down, last used Mon Jun  8 16:40:07 2015^cache.cpf^1972^57772^62972^^
		qlistStr := string(out)
		if qlistStr == "" {
			log.Printf("Error: Cannot continue as qlistStr from 'ccontrol qlist <instance>' is empty")
			os.Exit(1)
		}

		// parsing the returned string; they're all []string...
		upDownStr := strings.SplitN(qlistStr, "^", 4)

		// we are only interested in the 1st word: "running", "down" etc.
		CachéStatus := strings.SplitN(upDownStr[3], ",", 2)
		cstatus := CachéStatus[0]

		if cstatus == "running" {
			log.Printf("Caché started successfully\n")

		} else if cstatus == "down" {
			log.Printf("Caché stopped successfully\n")

		} else if cstatus == "sign-on inhibited" {
			log.Printf("Something is preventing Caché from starting in multi-user mode,\n")
			log.Printf("You might want to start the container with the flag -cstart=false to fix it.\n")

		} else {
			log.Printf("Un-recognized Caché status while trying to verify its '%s' status\n", what)
			log.Printf("-qlist string = %s.\n", qlistStr)
		}
	}

	return nil
}

// starting Caché app by calling a routine or class method
//
func startApp(inst string, nmsp string, rou string) (bool, error) {
	log.Printf("Starting app '%s' in '%s'...\n", rou, nmsp)

	cmd := "ccontrol"
	args := []string{"session", inst, "-U", nmsp, rou}
	c := exec.Command(cmd, args...)

	if dbg {
		log.Printf("cmd & args: %s, %q", cmd, args)
	}

	// preapring for stdout/stderr msg
	var out bytes.Buffer
	c.Stdout = &out

	if err := c.Run(); err != nil {
		errMsg := out.String()
		log.Printf("Error in launching routine %s in namespace %s:", rou, nmsp)
		log.Printf("Err: %s; %s", err, errMsg)
		os.Exit(1)
	}

	return true, nil
}

// Stopping Caché
//
func shutdownCaché(inst string) (bool, error) {
	log.Printf("Shutting down Caché...\n")

	cmd := "ccontrol"
	args := []string{"stop", inst, "quietly"}

	if err := exec.Command(cmd, args...).Run(); err != nil {
		log.Printf("Error & possible causes:\n")
		log.Printf("-wrong Caché instance name")
		log.Printf("-Caché up in single user mode (there was trouble @startup)")
		log.Printf("ERR: %s", err)
		os.Exit(1)
	}

	// check that the shutdown was successful
	if err := checkCmdOutcome("down", inst); err != nil {
		log.Printf("Error: Caché was not shutdown successfully.\n")
		os.Exit(1)
	}

	return true, nil
}

// check that we are on a =>3.16 kernel
// if not, set shmem size seg to shmem param
//
func checkKernelAndShmem(shmem int) error {
	var kVer float64
	var isLinuxType, isWindows bool

	// getting ready for clouds: Azure, AWS, GCP...
	switch runtime.GOOS {
	case "windows":
		if dbg {
			log.Printf("Checking kernel: it's Windows")
		}
		isWindows = true

	case "freebsd":
		if dbg {
			log.Printf("Checking kernel: it's freebsd")
		}
		isLinuxType = true

	case "linux":
		if dbg {
			log.Printf("Checking kernel: it's linux")
		}
		isLinuxType = true

	}

	if isLinuxType == true {

		// retrieve the kernel version
		ver, err := getKernelVersion()
		if err != nil {
			log.Printf("Error in checking kernel version")
			log.Printf("err: %s", err)
			os.Exit(1)
		} else {
			kVer = ver

			// if pre-3.16 we need to dynamically tune shmem
			if kVer < k316 {
				log.Printf("kernel version less than 3.16, auto-tuning it")

				// attempting to tune shmmax
				if _, err := setSharedMemSeg(shmem); err != nil {
					log.Printf("\nError setting shared memory: %s\n", err)
					os.Exit(1)
				}
			} else {
				//log.Printf("kernel version => 3.16; nothing to do.")
			}
		}

	} else if isWindows == true {
		log.Printf("Error: un-implemented")
	}

	return nil
}

// get the kernel version so that we know if we must tune it
//
func getKernelVersion() (float64, error) {
	var kVer float64
	cmd := "uname"
	args := []string{"-r"}

	c := exec.Command(cmd, args...)

	// organise to read the response form the bufferd IO
	var out bytes.Buffer
	c.Stdout = &out

	// and its execution
	if err := c.Run(); err != nil {
		log.Printf("Error & possible causes:\n")
		log.Printf("-insufficient privileges to run the uname command to find out the kernel version\n")
		log.Printf("-missing uname command in container\n")
		log.Printf("ERR: %s", err)
		os.Exit(1)
	} else {
		resp := out.String()
		if dbg {
			log.Printf("resp: %s\n", resp)
		}

		// Parsing the string from 'uname -r'
		// examples:
		// 3.8.0-19-generic		Bodhi Linux on Ubuntu
		// 3.10.0-123.20.1.el7.x86_64	SLES12
		// 3.10.0-123.el7.x86_64	RHEL7
		// 3.16.6-2-desktop		OpenSUSE
		// 3.10.0-123.20.1.el7.x86_64	CentOS
		// 3.16.0-34-generic		Ubuntu
		//
		// breaking up the linux kernel version string into its constituencies and
		// reforming the string with only the first 2 version numbers of
		// major.minor so that ParseFloat can accept it
		//
		// kvVls is a []string
		kvVls := strings.Split(resp, ".")
		var shortVer string = ""
		version := make([]string, 2)

		// extract exactly the first 2 values we need from the Split func
		for k, kvPiece := range kvVls {
			if dbg {
				log.Printf("k-%d) kvPiece=%s", k, kvPiece)
			}

			if k < 2 {
				version[k] = kvPiece // kvVls[k]
			}
		}

		if dbg {
			log.Printf("[]version = %q", version)
		}

		// join the strings (which are in the slice)
		shortVer = strings.Join(version, ".")
		kVer, err = strconv.ParseFloat(shortVer, 64)
		if err != nil {
			log.Printf("Error converting to float: %s\n", err)
			os.Exit(1)
		}

		if dbg {
			log.Printf("kVer: %f; %s", kVer, shortVer)
		}
	}

	return kVer, nil
}

// start eXtra service(s)
// launched as a goroutine so that bash or program does not hold our PID1 listening for SIGTERM
//
func startExtraService(exeCmd string, exeOK chan bool) {
	log.Printf("Starting eXtra service '%s' \n", exeCmd)

	cmdName := exeCmd

	// if user need to pass param to its shell it's preferable that they set environment variables
	// via the -e VAR1=VAL syntax when launching a container
	c := exec.Command(cmdName)

	// preparing for stdout/stderr msg
	var out bytes.Buffer
	c.Stdout = &out

	// with Start() there is no chance to monitor the return value...
	if err := c.Start(); err != nil {
		errMsg := out.String()
		log.Printf("Error in starting eXtra service: '%s'; Err: %s; %s\n", exeCmd, errMsg, err)
		//log.Printf("Err: %s; %s", errMsg, err)

		exeOK <- false

	} else {

		if dbg {
			log.Printf("exeOK == true")
		}

		// the call was OK
		// a user is advised to log error messages to the container logs
		exeOK <- true
	}
}

// Stopping eXtra service(s)
//
func stopExtraService(xstop string) (bool, error) {
	log.Printf("Shutting down eXtra service '%s'...\n", xstop)

	c := exec.Command(xstop)

	// preapring for stdout/stderr msg
	var out bytes.Buffer
	c.Stdout = &out

	if err := c.Run(); err != nil {
		errMsg := out.String()
		log.Printf("Error in stopping service(s) %s:", xstop)
		log.Printf("Err: %s; %s", errMsg, err)
		os.Exit(1)
	}

	return true, nil
}

// Caché container main
//
func main() {

	// flag handling
	pFinst := flag.String("i", "CACHE", "The Cachè instance name to start/stop")
	pFnmsp := flag.String("n", "", "The Caché application Namespace")
	pFrou := flag.String("r", "", "The Caché Routine name to start the app")
	pFstop := flag.Bool("cstop", true, "Allows container to avoid (false) Caché shutdown in case of throw-away containers")
	pFstart := flag.Bool("cstart", true, "Allows container to come up without (false) starting Caché or initialising shmem")
	pFnostu := flag.Bool("nostu", false, "Allows cstart to run with the nostu option for maintenance, single user access mode.")
	pFshmem := flag.Int("shmem", 512, "Shared Mem segment max size in MB; default value=512MB enough to install and play")
	pFlog := flag.Bool("cconsole", false, "Allows to show cconsole.log in current output.")

	// user option to start other services he might need (sshd, whatever...)
	pFexeStart := flag.String("xstart", "", "Allows startup eXecution of other services or processes via a single <myStart_shell_script.sh>")
	pFexeStop := flag.String("xstop", "", "Allows stop eXecution of other services or processes via a single <myStop_shell_script.sh>")

	pVersion := flag.Bool("version", false, "prints version")

	flag.Parse()

	inst := *pFinst
	nmsp := *pFnmsp
	rou := *pFrou
	cstop := *pFstop
	cstart := *pFstart
	nostu := *pFnostu
	shmem := *pFshmem
	cclog := *pFlog
	exeStart := *pFexeStart
	exeStop := *pFexeStop
	ver := *pVersion

	if dbg {
		log.Printf("flag instance: %s\n", inst)
		log.Printf("flag namesapce: %s\n", nmsp)
		log.Printf("flag routine: %s\n", rou)
		log.Printf("flag cstop: %t\n", cstop)
		log.Printf("flag cstart: %t\n", cstart)
		log.Printf("flag nostu: %t\n", nostu)
		log.Printf("flag shmem: %d\n", shmem)
		log.Printf("flag xstart: %s\n", exeStart)
		log.Printf("flag xstop: %s\n", exeStop)
		log.Printf("flag v: %s\n", ver)

		log.Printf("command supplied: %q\n", flag.Args())
		log.Printf("--\n")
	}

	// 0--
	// version__________________________________________________________
	if ver == true {
		fmt.Printf("ccontainermain Version %s\n", version)
		os.Exit(0)
	}

	// 1--
	// starting up Caché services_______________________________________
	//
	if cstart == true {

		// 1.1--
		// check linux kernel version and if necessary tune shmmax seg
		if err := checkKernelAndShmem(shmem); err != nil {
			log.Printf("Error in checking kernel version")
			log.Printf("ERR: %s", err)
			os.Exit(1)
		}

		// 1.2--
		// starting Caché
		_, err := startCaché(inst, nostu, cclog)
		if err != nil {
			log.Printf("\nError starting up Caché: %s\n", err)
			os.Exit(1)
		} else {
			log.Printf("Caché is up.\n")
		}

		// 1.3--
		// starting Caché app if we were told to
		if rou != "" && nmsp != "" {
			_, err = startApp(inst, nmsp, rou)
			if err != nil {
				log.Printf("\nError starting up the app %s in namespace %s; Err: %s\n", rou, nmsp, err)
				os.Exit(1)
			} else {
				log.Printf("App is up.\n")
			}
		}

	} else {
		// no point trying to bring it down if it was never started
		cstop = false
	}

	// 2--
	// allow other services to start
	//
	if exeStart != "" {
		var exeOK bool

		// unbuffered blocking channel
		chExeCheck := make(chan bool)
		go startExtraService(exeStart, chExeCheck)

		exeOK = <-chExeCheck
		if exeOK != true {
			log.Printf("Error in Starting eXtra service '%s'", exeStart)
		} else {
			log.Printf("eXtra service is up.")
		}
	}

	// 3--
	// signal trapping
	// buffered un-blocking channel
	//
	// we must use a buffered channel or risk missing the signal if we're not
	// ready to receive when the signal is sent.
	//
	cSig := make(chan os.Signal, 1)

	// checking for the most common interrupt signals
	signal.Notify(cSig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGHUP)

	// Block until a signal is received_____________
	sig := <-cSig
	log.Printf("Signal trapped: %s; %d\n", sig, sig)

	// if SIG*... received then run shutdown

	// 4--
	// Bring Caché down cleanly
	//
	if cstop == true {
		_, err := shutdownCaché(inst)
		if err != nil {
			log.Printf("\nError shutting down Caché: %s\n", err)
			os.Exit(1)
		} else {
			log.Printf("Caché is down.\n")
		}
	}

	// 5--
	// bring down the extra service(s).
	//
	if exeStop != "" {
		_, err := stopExtraService(exeStop)
		if err != nil {
			log.Printf("\nError shutting down eXtra service: %s; err: \n", exeStop, err)
			os.Exit(1)
		} else {
			log.Printf("eXtra service down.\n")
		}
	}
}
