package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

/*
 * Wrapper for doing the open-vpn dance for aws vpns that use SAML auth
 * Step 1: Attempt an openvpn connection using static names; that will give us the saml url
 * Step 2: Visit the SAML url in a web browser and catch the response
 * Step 3: Re-run openvpn with the new auth
 */

func main() {
	sourceConfigFile := flag.String("config", "~/.awsvpn.conf", "Source aws vpn config file")
	flag.Parse()
	configFilename, serverURL, serverPort, err := createTempConfigFile(*sourceConfigFile)
	ips, err := net.LookupIP("dns." + serverURL) // have to use "random" subdomain
	if err != nil || len(ips) == 0 {
		fmt.Fprintf(os.Stderr, "Could not get IPs for VPN server : %v\n", err)
		os.Exit(1)
	}

	serverURL = ips[0].String()
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(configFilename)
	fmt.Printf("Starting vpn to %s:%s\n", serverURL, serverPort)
	//Connect once to find the saml auth url to use
	samlAuthpage, sid, err := initalcontactFindSAMLURL(configFilename, serverURL, serverPort)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Opening webpage to auth now", samlAuthpage)
	openbrowser(samlAuthpage)
	a := newSAMLAuth(sid, serverURL, serverPort, configFilename)
	a.runHTTPServer()
}

type SAMLAuth struct {
	samlResponseChan chan string
	sidID            string
	server           string
	port             string
	confpath         string
}

func newSAMLAuth(sid, server, port, confpath string) *SAMLAuth {
	s := &SAMLAuth{samlResponseChan: make(chan string, 2), sidID: sid, server: server, port: port, confpath: confpath}
	return s
}
func (s *SAMLAuth) runHTTPServer() {
	go s.worker()
	http.HandleFunc("/", s.handleSAMLServer)
	log.Printf("Starting HTTP server at 127.0.0.1:35001")
	http.ListenAndServe("127.0.0.1:35001", nil)
}

func (s *SAMLAuth) worker() {
	//Listens for events from saml http server and spawns openvpn as appropriate
	for {
		select {
		case auth, ok := <-s.samlResponseChan:
			if !ok {
				return
			}
			//we have authentication, lets spawn the correct openvpn
			fmt.Println("Starting the actual openvpn ")
			runOpenVPNAuthenticated(auth, s.sidID, s.server, s.port, s.confpath)

		}
	}
}
func (s *SAMLAuth) handleSAMLServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		SAMLResponse := r.FormValue("SAMLResponse")
		if len(SAMLResponse) == 0 {
			log.Printf("SAMLResponse field is empty or not exists")
			return
		}
		s.samlResponseChan <- url.QueryEscape(SAMLResponse)
		log.Printf("Got SAMLResponse field")
		return
	default:
		fmt.Fprintf(w, "Error: POST method expected, %s recieved", r.Method)
	}
}

func runOpenVPNAuthenticated(samlAuth, sid, server, serverPort, confpath string) {
	fmt.Printf("Running openvpn with SID:%s server %s:%s\n", sid, server, serverPort)
	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.password")
	if err != nil {
		return
	}
	commandInput := fmt.Sprintf("%s\r\nCRV1::%s::%s\r\n", "N/A", sid, samlAuth)
	destFile.WriteString(commandInput)
	destFile.Close()

	cmd := exec.Command("sudo", "./openvpn-patched", "--config", confpath, "--remote", server, serverPort, "--auth-user-pass", destFile.Name())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Start()
	cmd.Wait()
	cmd.Process.Kill()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func initalcontactFindSAMLURL(confpath, server, serverPort string) (SAMLString, sid string, err error) {
	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.password")
	if err != nil {
		return
	}
	commandInput := fmt.Sprintf("%s\r\n%s\r\n", "N/A", "ACS::35001")
	destFile.WriteString(commandInput)
	destFile.Close()

	cmd := exec.Command("./openvpn-patched", "--config", confpath, "--remote", server, serverPort, "--auth-user-pass", destFile.Name())
	var outb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	cmd.Wait()
	// We wait until we get response
	scanner := bufio.NewScanner(&outb)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "AUTH_FAILED,CRV1") {
			//This line is our saml url
			parts := strings.Split(line, "https://")
			SAMLString = "https://" + parts[1]
			parts = strings.Split(line, ":")
			sid = parts[6]
		}
	}
	cmd.Process.Kill()
	return
}

//createTempConfigFile Creates a temporary config to use for authentication that has the server name parsed out
// and this is returned seperately
func createTempConfigFile(sourceFilePath string) (outputFilename string, server string, port string, err error) {
	//Read the source file in and strip the server path and port out, and copy to a temp file
	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.conf")
	if err != nil {
		return
	}
	outputWriter := bufio.NewWriter(destFile)
	defer outputWriter.Flush()
	defer destFile.Close()
	outputFilename = destFile.Name()
	//Read the source file in and copy all lines 1:1 except stripping server
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return
	}
	defer sourceFile.Close()

	scanner := bufio.NewScanner(sourceFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "auth-retry") {
			//Strip
		} else if strings.HasPrefix(line, "resolv-retry") {
			//Strip
		} else if strings.HasPrefix(line, "auth-federate") {
			//Strip
		} else if strings.HasPrefix(line, "remote ") {
			// Split this apart to find the hostname and the port
			fields := strings.Fields(line)
			server = fields[1]
			port = fields[2]

		} else {
			_, err = outputWriter.WriteString(line + "\n")
			if err != nil {
				return
			}
		}
	}
	outputWriter.Flush()
	if err = scanner.Err(); err != nil {
		return
	}
	return
}
