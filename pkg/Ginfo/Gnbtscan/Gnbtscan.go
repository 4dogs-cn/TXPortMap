package Gnbtscan

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

func Scan(ip string) (string, error) {
	payload := []byte{
		// see https://blog.skullsecurity.org/2009/nbstatnse-a-replacement-for-nbtscan-and-others
		0x13, 0x37, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x43, 0x4b,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
		0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41,
		0x00, 0x00, 0x21, 0x00, 0x01,
	}

	conn, err := net.Dial("udp", fmt.Sprintf("%s:137", ip))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(payload); err != nil {
		return "", err
	}

	buffer := make([]byte, 256)
	bufferLen, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	if bufferLen < 12 {
		return "", fmt.Errorf("invalid header")
	}

	body := buffer[:bufferLen]
	if body[6] == byte(0x00) && body[7] == byte(0x00) {
		return "", fmt.Errorf("no answer to our request")
	}

	body = body[12:] // remove header
	offset := 0
	for body[offset] != 0 {
		offset++
		if offset == len(body) {
			return "", fmt.Errorf("invalid payload")
		}
	}

	body = body[offset+1:]
	if len(body) < 12 {
		return "", fmt.Errorf("no answer to our request")
	}

	nameCnt := body[10]
	if nameCnt == 0 {
		return "", fmt.Errorf("no names available")
	}

	offset = 0
	names := body[11:]
	for names[offset] != 0 {
		offset++
		if offset == len(names) {
			break
		}
	}
	return string(bytes.TrimSpace(names[:offset])), nil
}
