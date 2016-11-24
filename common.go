package obinex

// WatchDir is the directory watched by obinex
const WatchDir = "/proj/i4obinex/"

// EndMarker is used to find the end of hw output
const EndMarker = "Graceful shutdown initiated"

// ControlHosts contains the mapping of buddy hostname to hardware box hostname.
var ControlHosts map[string]string = map[string]string{
	"faui49jenkins12": "faui49big01",
	"faui49jenkins13": "faui49big02",
	"faui49jenkins14": "faui49big03",
	"faui49jenkins15": "fastbox",
	"faui49jenkins21": "faui49jenkins25",
}
