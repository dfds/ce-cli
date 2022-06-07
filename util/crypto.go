package util

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
)

func GetCertificateSHAThumbprint(server *string, port uint) []string {

	// declare slice of strings to return values
	var returnData []string

	// establish remote connection
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", *server, port), &tls.Config{})
	if err != nil {
		fmt.Printf("Host: %s\n", *server)
		fmt.Printf("Port: %d\n", port)
		panic("failed to connect: " + err.Error())
	}

	// get all certificates
	allCert := conn.ConnectionState().PeerCertificates

	// get last certificate
	cert := allCert[len(allCert)-1]

	// get the sha1 fingerprint
	fingerprint := sha1.Sum(cert.Raw)

	// build fingerprint into a string
	var buf bytes.Buffer
	for _, f := range fingerprint {
		fmt.Fprintf(&buf, "%02X", f)
	}

	// close connection
	conn.Close()

	// add to slice
	returnData = append(returnData, buf.String())

	// return the slice
	return returnData
}
