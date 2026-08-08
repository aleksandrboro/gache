package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Parser struct {
	reader *bufio.Reader
}

func NewParser(r *bufio.Reader) *Parser {
	return &Parser{reader: r}
}

func (p *Parser) Parse() (RESPValue, error) {
	firstByte, err := p.reader.ReadByte()
	if err != nil {
		return RESPValue{}, err
	}

	switch firstByte {
	case '+':
		return p.parseSimpleString()
	case '-':
		return p.parseError()
	case ':':
		return p.parseInteger()
	case '$':
		return p.parseBulkString()
	case '*':
		return p.parseArray()
	default:
		return p.parseInline(firstByte)
	}
}

func (p *Parser) parseSimpleString() (RESPValue, error) {
	str, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	return RESPValue{
		Type: SimpleString,
		Str:  str,
	}, nil
}

func (p *Parser) parseError() (RESPValue, error) {
	message, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	return RESPValue{
		Type: Error,
		Str:  message,
	}, nil
}

func (p *Parser) parseInteger() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	num, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid integer: %s", line)
	}

	return RESPValue{
		Type: Integer,
		Int:  num,
	}, nil
}

func (p *Parser) parseBulkString() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	length, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid lenght: %s", line)
	}

	if length == -1 {
		return RESPValue{Type: BulkString, IsNull: true}, nil
	}

	if length < -1 {
		return RESPValue{}, fmt.Errorf("invalid bulk string length: %d", length)
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(p.reader, buf)
	if err != nil {
		return RESPValue{}, err
	}

	crlf := make([]byte, 2)
	_, err = io.ReadFull(p.reader, crlf)
	if err != nil {
		return RESPValue{}, err
	}

	if crlf[0] != '\r' || crlf[1] != '\n' {
		return RESPValue{}, errors.New("bulk string not terminated with CRLF")
	}

	return RESPValue{
		Type: BulkString,
		Str:  string(buf),
	}, nil
}

func (p *Parser) parseArray() (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	count, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid array count: %s", line)
	}

	if count == -1 {
		return RESPValue{Type: Array, IsNull: true}, nil
	}

	if count < -1 {
		return RESPValue{}, fmt.Errorf("invalid array count: %d", count)
	}

	if count == 0 {
		return RESPValue{Type: Array, Array: []RESPValue{}}, nil
	}

	elements := make([]RESPValue, count)
	for i := range count {
		elements[i], err = p.Parse()
		if err != nil {
			return RESPValue{}, err
		}
	}

	return RESPValue{
		Type:  Array,
		Array: elements,
	}, nil
}

func (p *Parser) parseInline(firstByte byte) (RESPValue, error) {
	line, err := p.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	fullLine := string(firstByte) + line
	parts := strings.Fields(fullLine)

	if len(parts) == 0 {
		return RESPValue{}, errors.New("empty inline command")
	}

	elements := make([]RESPValue, len(parts))
	for i, part := range parts {
		elements[i] = RESPValue{
			Type: BulkString,
			Str:  part,
		}
	}

	return RESPValue{
		Type:  Array,
		Array: elements,
	}, nil
}

func (p *Parser) readLine() (string, error) {
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return "", errors.New("invalid line ending")
	}

	return line[:len(line)-2], nil
}
