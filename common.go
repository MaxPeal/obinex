package obinex

// ControlHosts contains the mapping of buddy hostname to hardware box hostname.
var ControlHosts map[string]string = map[string]string{
	"faui49jenkins12": "faui49big01",
	"faui49jenkins13": "faui49big02",
	"faui49jenkins14": "faui49big03",
	"faui49jenkins21": "faui49jenkins25",
	"faui49bello2":    "fastbox",
}

// Broadcast enables multiple reads from a channel.
// Subscribe by sending a channel into the returned Channel. The subscribed
// channel will now receive all messages sent into the original channel.
func Broadcast(c chan string) chan<- chan string {
	cNewChans := make(chan chan string)
	go func() {
		cs := make([]chan string, 5)
		for {
			select {
			case newChan := <-cNewChans:
				cs = append(cs, newChan)
			case e := <-c:
				for _, outC := range cs {
					// send non-blocking to avoid one
					// channel breaking the whole
					// broadcast
					select {
					case outC <- e:
						break
					default:
						break
					}
				}
			}
		}
	}()
	return cNewChans
}
