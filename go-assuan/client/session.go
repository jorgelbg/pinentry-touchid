package client

import (
	"bufio"
	"errors"
	"io"
	"os/exec"

	"github.com/foxcpp/go-assuan/common"
)

// Session struct is a wrapper which represents an alive connection between
// client and server.
//
// In Assuan protocol roles of peers after handleshake is not same, for this
// reason there is no generic Session object that will work for both client and
// server. In pracicular, client.Session (the struct you are looking at)
// represents client side of connection.
type Session struct {
	Pipe    io.ReadWriteCloser
	Scanner *bufio.Scanner
}

// ReadWriteCloser - a bit of glue between io.ReadCloser and io.WriteCloser.
type ReadWriteCloser struct {
	io.ReadCloser
	io.WriteCloser
}

// Close closes both io.ReadCloser and io.WriteCloser. Writer will not closed
// if Reader close failed.
func (rwc ReadWriteCloser) Close() error {
	if err := rwc.ReadCloser.Close(); err != nil {
		return err
	}
	return rwc.WriteCloser.Close()
}

// Implements no-op Close() function in additional to holding reference to
// Reader and Writer.
type nopCloser struct {
	io.ReadWriter
}

func (clsr nopCloser) Close() error {
	return nil
}

// InitNopClose initiates session using passed Reader/Writer and NOP closer.
func InitNopClose(pipe io.ReadWriter) (*Session, error) {
	ses := &Session{nopCloser{pipe}, bufio.NewScanner(pipe)}
	ses.Scanner.Buffer(make([]byte, common.MaxLineLen), common.MaxLineLen)

	// Take server's OK from pipe.
	_, _, err := common.ReadLine(ses.Scanner)
	if err != nil {
		Logger.Println("... I/O error:", err)
		return nil, err
	}

	return ses, nil
}

// Init initiates session using passed Reader/Writer.
func Init(pipe io.ReadWriteCloser) (*Session, error) {
	Logger.Println("Starting session...")
	ses := &Session{pipe, bufio.NewScanner(pipe)}
	ses.Scanner.Buffer(make([]byte, common.MaxLineLen), common.MaxLineLen)

	// Take server's OK from pipe.
	_, _, err := common.ReadLine(ses.Scanner)
	if err != nil {
		Logger.Println("... I/O error:", err)
		return nil, err
	}

	return ses, nil
}

// InitCmd initiates session using command's stdin and stdout as a I/O channel.
// cmd.Start() will be done by this function and should not be done before.
func InitCmd(cmd *exec.Cmd) (*Session, error) {
	// Errors generally should not happen here but let's be pedantic because we are library.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		Logger.Println("Failed to start command ("+cmd.Path+"):", err)
		return nil, err
	}

	return Init(ReadWriteCloser{stdout, stdin})
}

// Close sends BYE and closes underlying pipe.
func (ses *Session) Close() error {
	Logger.Println("Closing session (sending BYE)...")
	if err := common.WriteLine(ses.Pipe, "BYE", ""); err != nil {
		Logger.Println("... I/O error:", err)
		return err
	}
	// Server should respond with "OK" , but we don't care.
	return ses.Pipe.Close()
}

// Reset sends RESET command.
// According to Assuan documentation: Reset the connection but not any existing
// authentication. The server should release all resources associated with the
// connection.
func (ses *Session) Reset() error {
	Logger.Println("Resetting session...")
	if err := common.WriteLine(ses.Pipe, "RESET", ""); err != nil {
		return err
	}
	// Take server's OK from pipe.
	ok, params, err := common.ReadLine(ses.Scanner)
	if err != nil {
		Logger.Println("... I/O error:", err)
		return err
	}
	if ok == "ERR" {
		Logger.Println("... Received ERR: ", params)
		return common.DecodeErrCmd(params)
	}
	if ok != "OK" {
		return errors.New("not 'ok' response")
	}
	return nil
}

// SimpleCmd sends command with specified parameters and reads data sent by server if any.
func (ses *Session) SimpleCmd(cmd string, params string) (data []byte, err error) {
	Logger.Println("Sending command:", cmd, params)
	err = common.WriteLine(ses.Pipe, cmd, params)
	if err != nil {
		Logger.Println("... I/O error:", err)
		return []byte{}, err
	}

	for {
		scmd, sparams, err := common.ReadLine(ses.Scanner)
		if err != nil {
			Logger.Println("... I/O error:", err)
			return []byte{}, err
		}

		if scmd == "OK" {
			return data, nil
		}
		if scmd == "ERR" {
			Logger.Println("... Received ERR: ", sparams)
			return []byte{}, common.DecodeErrCmd(sparams)
		}
		if scmd == "D" {
			data = append(data, []byte(sparams)...)
		}
	}
}

// Transact sends command with specified params and uses byte arrays in data
// argument to answer server's inquiries. Values in data can be either []byte
// or pointer to implementer of io.Reader.
func (ses *Session) Transact(cmd string, params string, data map[string]interface{}) (rdata []byte, err error) {
	Logger.Println("Initiating transaction:", cmd, params)
	err = common.WriteLine(ses.Pipe, cmd, params)
	if err != nil {
		return []byte{}, err
	}

	for {
		scmd, sparams, err := common.ReadLine(ses.Scanner)
		if err != nil {
			return []byte{}, err
		}

		if scmd == "INQUIRE" {
			inquireResp, prs := data[sparams]
			if !prs {
				Logger.Println("... unknown request:", sparams)
				if err := common.WriteLine(ses.Pipe, "CAN", ""); err != nil {
					return nil, err
				}

				// We asked for FOO but we don't have FOO.
				return []byte{}, errors.New("missing data with keyword " + sparams)
			}

			switch inquireResp.(type) {
			case []byte:
				if err := common.WriteData(ses.Pipe, inquireResp.([]byte)); err != nil {
					Logger.Println("... I/O error:", err)
					return []byte{}, err
				}
			case io.Reader:
				if err := common.WriteDataReader(ses.Pipe, inquireResp.(io.Reader)); err != nil {
					Logger.Println("... I/O error:", err)
					return []byte{}, err
				}
			default:
				return nil, errors.New("invalid type in data map value")
			}

			if err := common.WriteLine(ses.Pipe, "END", ""); err != nil {
				Logger.Println("... I/O error:", err)
				return []byte{}, err
			}
		}

		// Same as SimpleCmd.
		if scmd == "OK" {
			return rdata, nil
		}
		if scmd == "ERR" {
			Logger.Println("... Received ERR: ", sparams)
			return []byte{}, common.DecodeErrCmd(sparams)
		}
		if scmd == "D" {
			Logger.Println("... Received data chunk")
			rdata = append(rdata, []byte(sparams)...)
		}
	}
}

// Option sets options for connections.
func (ses *Session) Option(name string, value string) error {
	Logger.Println("Setting option", name, "to", value+"...")
	err := common.WriteLine(ses.Pipe, "OPTION", name+" = "+value)
	if err != nil {
		Logger.Println("... I/O error: ", err)
		return err
	}

	cmd, sparams, err := common.ReadLine(ses.Scanner)
	if err != nil {
		Logger.Println("... I/O error: ", err)
		return err
	}
	if cmd == "ERR" {
		Logger.Println("... Received ERR: ", sparams)
		return common.DecodeErrCmd(sparams)
	}

	return nil
}
