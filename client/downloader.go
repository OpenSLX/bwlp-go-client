package client

import (
	"log"
	"net"
	"io"
	"bufio"
	"strconv"
	"encoding/binary"
	"fmt"
	"github.com/OpenSLX/bwlp-go-client/bwlp"
)

type Downloader struct {
	data []byte
	conn net.Conn
	ti *bwlp.TransferInformation
	connReader *bufio.Reader
	connWriter *bufio.Writer

	fileSize int64
	chunkSize int64
	startOffset int64
	endOffset int64
	totalRead int64
}

func NewDownloader(hostname string, ti *bwlp.TransferInformation, imageVersion *bwlp.ImageVersionDetails) *Downloader {
	// initialize connection
	conn, err := net.Dial("tcp", hostname + ":" + strconv.FormatInt(int64(ti.PlainPort), 10))
	if err != nil {
		log.Printf("Error establishing connection: %s\n", err)
		return nil
	}
	// init reader and writer
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	if reader == nil || writer == nil {
		log.Printf("Error initializing reader/writer for transfer.\n")
		return nil
	}
	// init transfer by sending 'D'
	err = binary.Write(writer, binary.BigEndian, []byte("D"))
	if err != nil {
		log.Printf("Error sending download initiator: %s\n", err)
		return nil
	}
	// send token
	if err := sendKeyValue(writer, "TOKEN", string(ti.Token)); err != nil {
		log.Printf("Failed to send token: %s\n", err)
		return nil
	}
	var chunkSize int64 = 16 * 1024 * 1024 // 16MB
	d := Downloader{
		conn: conn,
		ti: ti,
		connReader: reader,
		connWriter: writer,
		fileSize: imageVersion.FileSize,
		chunkSize: chunkSize,
		startOffset: 0,
		endOffset: chunkSize,
		totalRead: 0,
	}
	return &d
}

func sendEndOfMeta(writer *bufio.Writer) (error) {
	// End of meta consists of two null bytes
	if err := binary.Write(writer, binary.BigEndian, []byte{0x00, 0x00}); err != nil {
		log.Printf("Error writing terminating sequence!")
		return err
	}
	writer.Flush()
	return nil
}

func sendKeyValue(writer *bufio.Writer, key string, value string) (error) {
	if len(key) <= 0 {
		return fmt.Errorf("Empty key!")
	}
	msg := key + "=" + value
	// To comply to java's readUtf8 method, we need to prepend the message
	// with the byte-encoded length of the message
	if err := binary.Write(writer, binary.BigEndian, int16(len(msg))); err != nil {
		log.Printf("Failed to write length!")
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, []byte(msg)); err != nil {
		log.Println("Failed to write payload!", err)
		return err
	}
	return sendEndOfMeta(writer)
}

func readMetaData(reader *bufio.Reader) (string, error) {
	// first 2 bytes in java's modified UTF-8 contain the length
	metaLengthAsBytes := make([]byte, 2)
	if err := binary.Read(reader, binary.BigEndian, &metaLengthAsBytes); err != nil {
		log.Printf("Failed to read meta message length: %s\n", err)
		return "", err
	}
	metaLength := binary.BigEndian.Uint16(metaLengthAsBytes)
	metaBytes := make([]byte, metaLength)
	if err := binary.Read(reader, binary.BigEndian, metaBytes); err != nil {
		log.Printf("Failed to read actual meta message: %s\n", err)
		return "", err
	}
	if err := readEndOfMeta(reader); err != nil {
		log.Printf("%s", err)
		return "", err
	}
	return string(metaBytes[:]), nil
}

func readEndOfMeta(reader *bufio.Reader) error {
	readBytes := make([]byte, 2)
	if err := binary.Read(reader, binary.BigEndian, readBytes); err != nil {
		log.Println("Error reading terminating sequence!", err)
		return err
	}
	if len(readBytes) != 2 || readBytes[0] != 0x00 || readBytes[1] != 0x00 {
		return fmt.Errorf("Terminating sequence expected, but got: [% x]", readBytes)
	}
	return nil
}

func (d *Downloader) requestRange(startOffset int64, endOffset int64) error {
	// send range request
	rangeString := strconv.FormatInt(startOffset, 10) + ":" + strconv.FormatInt(endOffset, 10)
	if err := sendKeyValue(d.connWriter, "RANGE", rangeString); err != nil {
		return err
	}
	// read confirmation
	meta, err := readMetaData(d.connReader)
	if err != nil {
		log.Printf("Error reading range confirmation: %s\n", err)
		return err
	}
	// match?
	if meta != "RANGE=" + rangeString {
		return fmt.Errorf("Unexpected RANGE request response from server.")
	}
	return nil
}
// finally the read interface
func (d *Downloader) Read(p []byte) (n int, err error) {
	if d.totalRead == d.fileSize {
		sendKeyValue(d.connWriter, "DONE", "")
		return 0, io.EOF
	}
	// now we need to buffer bytes from the remote connection
	// before handing them out to the reader
	if d.totalRead <= d.startOffset {
		// no data cached, request from remote connection
		if err := d.requestRange(d.startOffset, d.endOffset); err != nil {
			return 0, err
		}
	}
	// now read data
	n, err = d.connReader.Read(p)
	if err != nil {
		if err != io.EOF {
			log.Printf("READ ERROR: %s\n", err)
		}
		return 0, err
	}
	d.totalRead += int64(n)
	// update offsets?
	if d.totalRead == d.endOffset {
		d.startOffset = d.endOffset
		if d.endOffset + d.chunkSize > d.fileSize {
			d.endOffset = d.fileSize
		} else {
			d.endOffset = d.startOffset + d.chunkSize
		}
	}
	return n, err
}
