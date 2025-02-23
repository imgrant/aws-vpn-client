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
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Default configuration paths
const (
	defaultOpenVPNConfigPath = "./config.ovpn"
	defaultOpenVPNBinaryPath = "openvpn"
)

// Re-assignable variables to allow build-time overrides
var (
	openVpnConfigFile = defaultOpenVPNConfigPath
	openVpnBinary = defaultOpenVPNBinaryPath
)

// awsSAMLAuthWrapper handles the AWS VPN SAML authentication process
type awsSAMLAuthWrapper struct {
	reauthRequest    chan bool
	samlResponseChan chan string
	sidID            string
	server           string
	port             string
	confPath         string
	openVpnCmd       string
}

func main() {
	sourceConfigFile := flag.String("config", openVpnConfigFile, "Path to the OpenVPN config file")
	openVpnCmd := flag.String("openvpn", openVpnBinary, "Path to the AWS-patched OpenVPN binary")
	flag.Parse()

	filePath := expandHomeDir(*sourceConfigFile)
	fmt.Println("Loading config from", filePath)

	configFilename, serverURL, serverPort, err := createTempConfigFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(configFilename)

	serverURL, err = resolveServerURL(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting VPN connection to %s:%s\n", serverURL, serverPort)
	auth := newAWSSAMLAuthWrapper(serverURL, serverPort, configFilename, *openVpnCmd)
	auth.runHTTPServer()
}

func newAWSSAMLAuthWrapper(server, port, confpath, openVpnCmd string) *awsSAMLAuthWrapper {
	return &awsSAMLAuthWrapper{
		samlResponseChan: make(chan string, 2),
		reauthRequest:    make(chan bool, 2),
		server:           server,
		port:             port,
		confPath:         confpath,
		openVpnCmd:       openVpnCmd,
	}
}

func (s *awsSAMLAuthWrapper) runHTTPServer() {
	go s.worker()
	s.reauthRequest <- true // Initiate authentication process

	http.HandleFunc("/", s.handleSAMLServer)
	log.Printf("Starting HTTP server at 127.0.0.1:35001")
	err := http.ListenAndServe("127.0.0.1:35001", nil)
	log.Fatal(err)
}

func (s *awsSAMLAuthWrapper) worker() {
	for {
		select {
		case auth, ok := <-s.samlResponseChan:
			if !ok {
				return
			}
			fmt.Println("Starting OpenVPN with authenticated credentials")
			runOpenVPNAuthenticated(auth, s.sidID, s.server, s.port, s.confPath, s.openVpnCmd)
			fmt.Println("OpenVPN exited unexpectedly. Re-authenticating...")
			s.stageOne()

		case <-s.reauthRequest:
			s.stageOne()
		}
	}
}

func (s *awsSAMLAuthWrapper) stageOne() {
	samlAuthpage, sid, err := initialContactFindSAMLURL(s.confPath, s.server, s.port, s.openVpnCmd)
	if err != nil {
		log.Fatal(err)
	}
	s.sidID = sid
	fmt.Println("Opening webpage for SAML authentication:", samlAuthpage)
	openBrowser(samlAuthpage)
}

func (s *awsSAMLAuthWrapper) handleSAMLServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "Error: POST method expected, %s received", r.Method)
		return
	}

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		log.Printf("SAMLResponse field is empty or doesn't exist")
		return
	}

	s.samlResponseChan <- url.QueryEscape(samlResponse)
	log.Printf("Got SAMLResponse field")
}

func runOpenVPNAuthenticated(samlAuth, sid, server, serverPort, confpath, openVpnCmd string) {
	fmt.Printf("SID:%s server %s:%s\n", sid, server, serverPort)

	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.password")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(destFile.Name())

	commandInput := fmt.Sprintf("%s\r\nCRV1::%s::%s\r\n", "N/A", sid, samlAuth)
	if err := ioutil.WriteFile(destFile.Name(), []byte(commandInput), 0600); err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.Command("sudo", openVpnCmd, "--config", confpath,
		"--remote", server, serverPort,
		"--auth-user-pass", destFile.Name())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Channel to track if we're shutting down due to a signal
	signalShutdown := make(chan struct{})

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		if cmd.Process != nil {
			fmt.Println("Forwarding signal to OpenVPN process...")
			cmd.Process.Signal(sig)
			close(signalShutdown)
		}
	}()

	// Wait for OpenVPN to exit
	err = cmd.Wait()

	// The OpenVPN process has finished, give it a moment to priny any cleanup messages
	time.Sleep(100 * time.Millisecond)

	// Check if we're shutting down due to a signal
	select {
	case <-signalShutdown:
		// We sent a signal to OpenVPN, exit the program
		os.Exit(0)
	default:
		// OpenVPN exited on its own
		if err != nil {
			// Only return (which triggers re-authentication) if OpenVPN wasn't terminated by a signal
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						// Exit if it was terminated by a signal
						os.Exit(0)
					}
				}
			}
		}
		// If we get here, OpenVPN exited unexpectedly, so return to allow re-authentication
		return
	}
}

func openBrowser(url string) {
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

func initialContactFindSAMLURL(confpath, server, serverPort, openVpnCmd string) (samlString, sid string, err error) {
	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.password")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(destFile.Name())

	commandInput := fmt.Sprintf("%s\r\n%s\r\n", "N/A", "ACS::35001")
	if err := ioutil.WriteFile(destFile.Name(), []byte(commandInput), 0600); err != nil {
		return "", "", err
	}

	fmt.Println("Starting initial OpenVPN to get SAML URL and SID for authentication")
	cmd := exec.Command(openVpnCmd, "--config", confpath,
		"--remote", server, serverPort,
		"--auth-user-pass", destFile.Name())

	var outb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", "", err
	}

	if err := cmd.Wait(); err != nil {
		// Ignore error as we expect auth failure
	}

	scanner := bufio.NewScanner(&outb)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "AUTH_FAILED,CRV1") {
			parts := strings.Split(line, "https://")
			samlString = "https://" + parts[1]
			parts = strings.Split(line, ":")
			sid = parts[6]
			break
		}
	}

	return samlString, sid, scanner.Err()
}

func createTempConfigFile(sourceFilePath string) (outputFilename, server, port string, err error) {
	destFile, err := ioutil.TempFile("", "aws_vpn_wrapper_config_*.conf")
	if err != nil {
		return "", "", "", err
	}
	defer destFile.Close()

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return "", "", "", err
	}
	defer sourceFile.Close()

	writer := bufio.NewWriter(destFile)
	scanner := bufio.NewScanner(sourceFile)

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "auth-retry"),
			strings.HasPrefix(line, "resolv-retry"),
			strings.HasPrefix(line, "auth-federate"):
			continue
		case strings.HasPrefix(line, "remote "):
			fields := strings.Fields(line)
			server = fields[1]
			port = fields[2]
		default:
			if _, err := writer.WriteString(line + "\n"); err != nil {
				return "", "", "", err
			}
		}
	}

	if err := writer.Flush(); err != nil {
		return "", "", "", err
	}

	return destFile.Name(), server, port, scanner.Err()
}

func expandHomeDir(filePath string) string {
	if filePath == "~" {
		usr, _ := user.Current()
		return usr.HomeDir
	}
	if strings.HasPrefix(filePath, "~/") {
		usr, _ := user.Current()
		return filepath.Join(usr.HomeDir, filePath[2:])
	}
	return filePath
}

func resolveServerURL(serverURL string) (string, error) {
	ips, err := net.LookupIP("dns." + serverURL)
	if err != nil || len(ips) == 0 {
		return "", fmt.Errorf("could not get IPs for VPN server: %v", err)
	}
	return ips[0].String(), nil
}