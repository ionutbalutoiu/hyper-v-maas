package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

func execPsCmd(command string) ([]byte, error) {
	psCommand := fmt.Sprintf(`$ErrorActionPreference="Stop";try { %s } catch { Write-Host $_; exit 1 }`, command)
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", psCommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmd.ProcessState.ExitCode() != 0 {
			message := strings.TrimSpace(string(output))
			return []byte{}, errors.New(message)
		}
		return []byte{}, err
	}
	return output, nil
}

func powerOnMachine(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "only http post method is supported")
		return
	}
	machineName := r.URL.Query().Get("machine-name")
	if machineName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "machine-name query string parameter is required")
		return
	}
	bootOrderPsCmd := fmt.Sprintf(`Set-VMFirmware -VMName %[1]s -FirstBootDevice ((Get-VMFirmware -VMName %[1]s).BootOrder | ? BootType -eq Network)[0].Device`, machineName)
	if _, err := execPsCmd(bootOrderPsCmd); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
	startPsCmd := fmt.Sprintf("(Start-VM -Name %s).State", machineName)
	if _, err := execPsCmd(startPsCmd); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

func powerOffMachine(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "only http post method is supported")
		return
	}
	machineName := r.URL.Query().Get("machine-name")
	if machineName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "machine-name query string parameter is required")
		return
	}
	psCmd := fmt.Sprintf("Stop-VM %s -TurnOff -Force -Confirm:$false", machineName)
	if _, err := execPsCmd(psCmd); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
}

func queryMachinePowerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "only http get method is supported")
		return
	}
	machineName := r.URL.Query().Get("machine-name")
	if machineName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "machine-name query string parameter is required")
		return
	}
	psCmd := fmt.Sprintf("(Get-VM -Name %s).State", machineName)
	stdout, err := execPsCmd(psCmd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err.Error())
		return
	}
	fmt.Fprintln(w, strings.TrimSpace(string(stdout)))
}

func main() {
	listenAddress := flag.String("listen-address", "0.0.0.0:4321", "Listen address")

	http.HandleFunc("/power-on", powerOnMachine)
	http.HandleFunc("/power-off", powerOffMachine)
	http.HandleFunc("/power-status", queryMachinePowerStatus)

	fmt.Printf("Listening on %s\n", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		panic(err)
	}
}
