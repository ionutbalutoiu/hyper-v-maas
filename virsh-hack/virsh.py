#!/usr/bin/env python3

import sys
import subprocess

from threading import Timer


DEBUG_LOG_FILE = "/tmp/virsh-hyper-v-proxy.log"


def write_debug_log(message):
    with open(DEBUG_LOG_FILE, "a") as f:
        f.write("%s\n" % message)


def run_shell_cmd(cmd, cwd=None, env=None, timeout=60):

    def kill_proc_timout(proc):
        proc.kill()
        raise Exception("Timeout of %s seconds exceeded for cmd %s" % (
            timeout, cmd))

    f_stderr = subprocess.PIPE
    f_stdout = subprocess.PIPE
    proc = subprocess.Popen(cmd, env=env, cwd=cwd, shell=True,
                            stdout=f_stdout, stderr=f_stderr)
    timer = Timer(timeout, kill_proc_timout, [proc])

    try:
        timer.start()
        stdout, stderr = proc.communicate()
        if proc.returncode:
            raise Exception("Failed to execute cmd: %s. Exit code: %s" % (
                cmd, proc.returncode))
        if stdout:
            stdout = stdout.decode("ascii").strip()
        if stderr:
            stderr = stderr.decode("ascii").strip()
        return stdout, stderr
    finally:
        timer.cancel()


def get_vm_state(vm_name):
    state, _ = run_shell_cmd("ssh hyper-v '(Get-VM %s).State'" % vm_name)
    if state == "Running":
        return "running"
    return "shut off"


def start_vm(vm_name):
    network_devices = ("(Get-VMFirmware -VMName {0}).BootOrder | ? "
                       "BootType -eq Network".format(vm_name))
    boot_order_cmd = ("Set-VMFirmware -VMName {0} -FirstBootDevice "
                      "({1})[0].Device".format(vm_name, network_devices))
    run_shell_cmd("ssh hyper-v '{0}'".format(boot_order_cmd))
    run_shell_cmd("ssh hyper-v Start-VM {0}".format(vm_name))


def stop_vm(vm_name):
    run_shell_cmd("ssh hyper-v Stop-VM %s -TurnOff -Force -Confirm:\$false" % vm_name)


while True:
    user_input = input("virsh # ").split()
    # write_debug_log(user_input)

    if user_input[0] == "domstate":
        print(get_vm_state(user_input[1]))

    if user_input[0] == "start":
        start_vm(user_input[1])

    if user_input[0] == "destroy":
        stop_vm(user_input[1])
