# METADATA
# title: Insecure ports must not be exposed
# description: |
#   Blocks images that EXPOSE ports of protocols that are insecure or should
#   never be reachable from a container: FTP, SSH, Telnet, MS-RPC, NetBIOS,
#   SMB, unauthenticated Docker API, RDP and VNC.
#   Keep this list in sync with INSECURE_PORTS in .github/workflows/ci.yml.
# scope: package
# schemas:
#   - input: schema["dockerfile"]
# custom:
#   id: NP01-PORT-001
#   avd_id: NP01-PORT-001
#   severity: HIGH
#   short_code: no-insecure-ports
#   recommended_actions: Remove the EXPOSE instruction for this port; serve the application over a non-privileged application port instead.
#   input:
#     selector:
#       - type: dockerfile
package user.dockerfile.np01_insecure_ports

import rego.v1

insecure_ports := {
	21: "FTP",
	22: "SSH",
	23: "Telnet",
	135: "MS-RPC",
	137: "NetBIOS name service",
	138: "NetBIOS datagram service",
	139: "NetBIOS session service",
	445: "SMB",
	2375: "Docker API without TLS",
	3389: "RDP",
	5900: "VNC",
}

# EXPOSE accepts "80", "80/tcp", "137-139/udp".
port_range(value) := [lo, hi] if {
	spec := split(value, "/")[0]
	bounds := split(spec, "-")
	lo := to_number(bounds[0])
	hi := to_number(bounds[count(bounds) - 1])
}

deny contains res if {
	some stage in input.Stages
	some instr in stage.Commands
	instr.Cmd == "expose"
	some value in instr.Value
	[lo, hi] := port_range(value)
	some port, name in insecure_ports
	port >= lo
	port <= hi
	res := result.new(sprintf("Port %d (%s) is exposed; it is considered insecure", [port, name]), instr)
}
