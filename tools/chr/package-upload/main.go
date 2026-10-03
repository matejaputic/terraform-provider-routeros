// SPDX-License-Identifier: MPL-2.0
// Owned-loopback fixture provisioning only. Minimal bounded SFTP v3 upload;
// no shell execution, user SSH-key mutation, credential or packet logging.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

func word(b *bytes.Buffer, n uint32) { _ = binary.Write(b, binary.BigEndian, n) }
func text(b *bytes.Buffer, p []byte) { word(b, uint32(len(p))); b.Write(p) }
func packet(w io.Writer, kind byte, b *bytes.Buffer) error {
	head := new(bytes.Buffer)
	word(head, uint32(b.Len()+1))
	head.WriteByte(kind)
	head.Write(b.Bytes())
	_, e := w.Write(head.Bytes())
	return e
}
func response(r io.Reader) (byte, *bytes.Reader, error) {
	var n uint32
	if e := binary.Read(r, binary.BigEndian, &n); e != nil {
		return 0, nil, e
	}
	if n < 1 || n > 1<<20 {
		return 0, nil, errors.New("invalid SFTP response size")
	}
	data := make([]byte, n)
	if _, e := io.ReadFull(r, data); e != nil {
		return 0, nil, e
	}
	return data[0], bytes.NewReader(data[1:]), nil
}
func upload() error {
	if len(os.Args) != 3 {
		return errors.New("owned fixture credentials and pinned package path required")
	}
	data, e := os.ReadFile(os.Args[1])
	if e != nil {
		return e
	}
	creds := map[string]string{}
	if e = json.Unmarshal(data, &creds); e != nil {
		return e
	}
	if creds["hosturl"] != "http://127.0.0.1:18780" || creds["username"] != "admin" {
		return errors.New("not an owned loopback fixture")
	}
	file, e := os.Open(os.Args[2])
	if e != nil {
		return e
	}
	defer file.Close()
	name := filepath.Base(os.Args[2])
	if filepath.Ext(name) != ".npk" {
		return errors.New("package filename required")
	}
	conn, e := net.DialTimeout("tcp", "127.0.0.1:18722", 15*time.Second)
	if e != nil {
		return e
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(90 * time.Second))
	// Host identity is newly generated inside an explicitly owned disposable guest.
	sc, chans, requests, e := ssh.NewClientConn(conn, "127.0.0.1:18722", &ssh.ClientConfig{User: creds["username"], Auth: []ssh.AuthMethod{ssh.Password(creds["password"])}, HostKeyCallback: ssh.InsecureIgnoreHostKey()})
	if e != nil {
		return e
	}
	client := ssh.NewClient(sc, chans, requests)
	defer client.Close()
	session, e := client.NewSession()
	if e != nil {
		return e
	}
	defer session.Close()
	w, e := session.StdinPipe()
	if e != nil {
		return e
	}
	r, e := session.StdoutPipe()
	if e != nil {
		return e
	}
	if e = session.RequestSubsystem("sftp"); e != nil {
		return e
	}
	b := new(bytes.Buffer)
	word(b, 3)
	if e = packet(w, 1, b); e != nil {
		return e
	}
	kind, _, e := response(r)
	if e != nil || kind != 2 {
		return errors.New("SFTP negotiation failed")
	}
	b.Reset()
	word(b, 1)
	text(b, []byte(name))
	word(b, 2|8|16)
	word(b, 0)
	if e = packet(w, 3, b); e != nil {
		return e
	}
	kind, reply, e := response(r)
	if e != nil || kind != 102 {
		return errors.New("SFTP package open failed")
	}
	var id, size uint32
	binary.Read(reply, binary.BigEndian, &id)
	binary.Read(reply, binary.BigEndian, &size)
	if id != 1 || size > 65536 {
		return errors.New("invalid SFTP handle")
	}
	handle := make([]byte, size)
	if _, e = io.ReadFull(reply, handle); e != nil {
		return e
	}
	offset := uint64(0)
	id = 2
	chunk := make([]byte, 16384)
	for {
		n, readerr := file.Read(chunk)
		if n > 0 {
			b.Reset()
			word(b, id)
			text(b, handle)
			binary.Write(b, binary.BigEndian, offset)
			text(b, chunk[:n])
			if e = packet(w, 6, b); e != nil {
				return e
			}
			kind, reply, e = response(r)
			if e != nil || kind != 101 {
				return errors.New("SFTP write failed")
			}
			var returned, status uint32
			binary.Read(reply, binary.BigEndian, &returned)
			binary.Read(reply, binary.BigEndian, &status)
			if returned != id || status != 0 {
				return errors.New("SFTP write rejected")
			}
			offset += uint64(n)
			id++
		}
		if readerr == io.EOF {
			break
		}
		if readerr != nil {
			return readerr
		}
	}
	b.Reset()
	word(b, id)
	text(b, handle)
	if e = packet(w, 4, b); e != nil {
		return e
	}
	kind, reply, e = response(r)
	if e != nil || kind != 101 {
		return errors.New("SFTP close failed")
	}
	var returned, status uint32
	binary.Read(reply, binary.BigEndian, &returned)
	binary.Read(reply, binary.BigEndian, &status)
	if returned != id || status != 0 {
		return errors.New("SFTP close rejected")
	}
	return nil
}
func main() {
	if upload() != nil {
		os.Stderr.WriteString("owned package upload failed (credentials and protocol data suppressed)\n")
		os.Exit(1)
	}
}
