package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// MaxLineLen is a maximum length of line in Assuan protocol, including
	// space after command and LF.
	MaxLineLen = 1000
)

// ReadWriter ties arbitrary io.Reader and io.Writer to get a struct that
// satisfies io.ReadWriter requirements.
type ReadWriter struct {
	io.Reader
	io.Writer
}

// ReadLine reads raw request/response in following format: command <parameters>
//
// Empty lines and lines starting with # are ignored as specified by protocol.
// Additionally, status information is silently discarded for now.
func ReadLine(scanner *bufio.Scanner) (cmd string, params string, err error) {
	var line string
	for {
		if ok := scanner.Scan(); !ok {
			err := scanner.Err()
			if err == nil {
				err = io.EOF
			}
			return "", "", err
		}
		line = scanner.Text()

		// We got something that looks like a message. Let's parse it.
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "S ") && len(strings.TrimSpace(line)) != 0 {
			break
		}
	}

	// Part before first whitespace is a command. Everything after first whitespace is parameters.
	parts := strings.SplitN(line, " ", 2)

	// If there is no parameters... (huh!?)
	if len(parts) == 1 {
		return strings.ToUpper(parts[0]), "", nil
	}

	Logger.Println("<", parts[0])

	params, err = unescapeParameters(parts[1])
	if err != nil {
		return "", "", err
	}

	// Command is "normalized" to upper case since peer can send
	// commands in any case.
	return strings.ToUpper(parts[0]), params, nil
}

// WriteLine writes request/response to underlying pipe.
// Contents of params is escaped according to requirements of Assuan protocol.
func WriteLine(pipe io.Writer, cmd string, params string) error {
	if len(cmd)+len(params)+2 > MaxLineLen {
		Logger.Println("Refusing to send too long command")
		// 2 is for whitespace after command and LF
		return errors.New("too long command or parameters")
	}

	Logger.Println(">", cmd)

	line := []byte(strings.ToUpper(cmd) + " " + escapeParameters(params) + "\n")
	_, err := pipe.Write(line)
	return err
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// WriteData sends passed byte slice using one or more D commands.
// Note: Error may occur even after some data is written so it's better
// to just CAN transaction after WriteData error.
func WriteData(pipe io.Writer, input []byte) error {
	encoded := []byte(escapeParameters(string(input)))
	chunkLen := MaxLineLen - 3 // 3 is for 'D ' and line feed.
	for i := 0; i < len(encoded); i += chunkLen {
		chunk := encoded[i:min(i+chunkLen, len(encoded))]
		chunk = append([]byte{'D', ' '}, chunk...)
		chunk = append(chunk, '\n')

		if _, err := pipe.Write(chunk); err != nil {
			return err
		}
	}
	return nil
}

// WriteDataReader is similar to WriteData but sends data from input Reader
// until EOF.
func WriteDataReader(pipe io.Writer, input io.Reader) error {
	chunkLen := MaxLineLen - 3 // 3 is for 'D ' and line feed.
	buf := make([]byte, chunkLen)

	for {
		n, err := input.Read(buf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		chunk := []byte(escapeParameters(string(buf[:n])))
		chunk = append([]byte{'D', ' '}, chunk...)
		chunk = append(chunk, '\n')
		if _, err := pipe.Write(chunk); err != nil {
			return err
		}
	}
}

// ReadData reads sequence of D commands and joins data together.
func ReadData(scanner *bufio.Scanner) (data []byte, err error) {
	for {
		cmd, chunk, err := ReadLine(scanner)
		if err != nil {
			return nil, err
		}

		if cmd == "END" {
			return data, nil
		}

		if cmd == "CAN" {
			return nil, Error{ErrSrcAssuan, ErrUnexpected, "assuan", "IPC call has been cancelled"}
		}

		if cmd != "D" {
			return nil, Error{ErrSrcAssuan, ErrUnexpected, "assuan", "unexpected IPC command"}
		}

		unescaped, err := unescapeParameters(chunk)
		if err != nil {
			return nil, err
		}

		data = append(data, []byte(unescaped)...)
	}
}

// WriteComment is special case of WriteLine. "Command" is # and text is parameter.
func WriteComment(pipe io.Writer, text string) error {
	return WriteLine(pipe, "#", text)
}

func WriteError(pipe io.Writer, err Error) error {
	return WriteLine(pipe, "ERR", fmt.Sprintf("%d %s <%s>", MakeErrCode(err.Src, err.Code), err.Message, err.SrcName))
}
