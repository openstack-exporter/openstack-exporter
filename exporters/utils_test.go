package exporters

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"testing"
)

func TestAdditionalTLSTrust(t *testing.T) {
	var dummyPEM = `-----BEGIN CERTIFICATE-----
MIIFXTCCA0WgAwIBAgIUGNguxdFGdSAKvofW9qD2NDhH4lkwDQYJKoZIhvcNAQEL
BQAwPTELMAkGA1UEBhMCRFUxDjAMBgNVBAgMBUR1bW15MQ4wDAYDVQQKDAVEdW1t
eTEOMAwGA1UEAwwFZHVtbXkwIBcNMjIxMDA4MTIxNzM2WhgPMjA3MjA5MjUxMjE3
MzZaMD0xCzAJBgNVBAYTAkRVMQ4wDAYDVQQIDAVEdW1teTEOMAwGA1UECgwFRHVt
bXkxDjAMBgNVBAMMBWR1bW15MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKC
AgEA1nQjJdjxCt/q38msg8AfzBDbBW5XGENiqRbZXGXEFnXB2yB+s3Xaar9lxVHg
L9y53ObeV6M6M3FRh2jLZZypOZEqdptav2zpy/iE0Q9XtpYDfTz0wV50aERTLBNm
l0TLPab8Ee9MiZzb3Ysht1bcu7N+7UIDC2J5kOvT2791v23Q981wUgi79PBAzsXl
8NbTSr0KwvtWqlljLz2r2d4KWNuOBQZrFfqI1aMwbVWdeCE/SFubsxnl3wM2h6CB
8bSjkKi2L/Rd2xOKzMvl+K0Us0/M6WTUQL/KH8OC76B5ZS0iGfoyggsr1PpASweR
HkN2buMG0oC2NYiocldHP01BAKGYV2GTQBvYbqRGkznc3iADZKscA3Xgd4AVGQRE
aYImmiuUaGCIQw5JxWYJRe3HtMeUrbJ2FuqQ3D3+QSuMYw/UEUFVirRIyfpKIfVI
2ad7+JLKxnGe8WhC5yf2A764H6RLVwXKUoX7OJa8BinRl7VL6/JeZVx8VxzWKueg
pdlDvV5iHa4l9Vwg4ZgSGCeBh/9nKCWEWcUGPCzlR8I6BBmiwKrrOH+jDWcmMJ4I
bWe9XTNFXdPj/mrV+AEmbuui6kNIbUg4yGh1ZK7dqWsD9MzjdEbv+zfdab2PXeE/
C5lK0GoSU/NjEKp06Or9V3JOoEMgKX9g8WFiCKm3n/0hpC8CAwEAAaNTMFEwHQYD
VR0OBBYEFJHvMryFfHWFVck0q38THNsdn9PqMB8GA1UdIwQYMBaAFJHvMryFfHWF
Vck0q38THNsdn9PqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggIB
ABIE2BmhOjjca5TSUaXBL6P3yexPbhr62t0zwLOXhNYbnFG6foZ+B4VzhKyD84/k
PnXIs8XdJbIpUWrLEPPFHnanPLSY4sjspHtvuDIG3hhDUWAzBEIvKtv3Psb3wtgO
pEbDB1izqxLjKLK0+f3rTVEvw14TVTsDEj2mNhESPZTEWQUD55/grTlJ2OgtjQiY
4EwvPQ/0XgqFq/sJjsoqJDQZNH6iBFNguwgEyFBWBqDIe//MRVs+CwZCHSbbT0Is
piYLU5iLWK21nBXHqS+0hYEE8QAM+DdAEqAiqGQVZ9wS2dRBKur6aF9sqJ0GlBwj
RcrXgRnqCZTK4DWhRijY1cqi+HebyptkcFV6kNjGmug1dboUBqOyWG2dh9d+GhoK
dp1hB9uo+AuZ7viwpFB91VPp22QH7/YNNmn7PR6rtSyzL/fXGsrTg2JVKz3Pi4wK
jR/sZEGPpnbeyFj3sP/qMOKtV1lClPIP/Z0NQ5Z6AefDpcYePHxaQpJARRi5vDFG
b35h47EFeeXMgG8pOJuD4lPbn0VwvFfFEdANozStedCWjW8xf3zFVrAclUz0PKk7
gz3KbPLgOAo6Cza6lQsZR8a4r/FgoUPDQHmooMNDt2z6wsZTlWQH2P5sTfiFORZV
kf2kRqwmo4NpwI1Zb5eaQa6ca6qBaAQ35l+bpes7VEQX
-----END CERTIFICATE-----`

	// Get the SystemCertPool, continue with an empty pool on error to mimic target function behavior
	ourCertPool, err := x509.SystemCertPool()
	if ourCertPool == nil {
		fmt.Printf("Creating a new empty CertPool as we failed to load it from disk: %v\n", err)
		ourCertPool = x509.NewCertPool()
	}
	// Keep a untouched pool for comparison later
	untouchedSystemCertPool := ourCertPool.Clone()

	// Append our certificate
	ourCertPool.AppendCertsFromPEM(bytes.TrimSpace([]byte(dummyPEM)))

	// Append the passed certificate via the function we test
	certPool, err := additionalTLSTrust(dummyPEM, nil)
	if err != nil {
		t.Errorf("prepareTLSConfig failed with error: %s", err)
	}

	// Make sure we actually modified the CertPool at all
	if untouchedSystemCertPool.Equal(ourCertPool) {
		t.Errorf("Untouched SystemCertPool is equal to our supposedly modified CertPool")
	}

	// Our CertPool should match the returned CertPool
	if !ourCertPool.Equal(certPool) {
		t.Error("Cert pools are not equal")
	}
}
