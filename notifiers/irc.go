// IRC
//
// The IRC notification object announces test-failures to an IRC channel.
//
// Assuming you wish to post notification faiures to the room
// "#sysadmin", on the server "irc.example.com", as user
// "Bot" you'd set your connection string to:
//
//    irc://Bot@irc.example.com/#sysadmin
//
// If you wish to use TLS use "ircs":
//
//    ircs://Bot:password@irc.example.com/#sysadmin
//
// Finally if you need a password to join the server add ":password"
// appropriately.
//

package notifiers

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/skx/overseer/test"
	"github.com/thoj/go-ircevent"
)

// IRCNotifier is our object.
type IRCNotifier struct {
	// data is the URI describing the IRC server to connect to
	data string

	// ircconn holds the IRC server connection.
	irccon *irc.Connection

	// Have we joined our channel?
	joined bool

	// Record the channel name here, for sending the message
	channel string

	// Avoid threading issues with our passed/failed counts
	mutex sync.RWMutex

	// Count of how many tests have executed and passed
	passed int64

	// Count of how many tests have executed and failed
	failed int64
}

// Setup connects to the IRC server which was mentioned in the
// data passed to the constructor.
func (s *IRCNotifier) Setup() error {

	//
	// Parse the configuration URL
	//
	u, err := url.Parse(s.data)
	if err != nil {
		return err
	}

	//
	// Get fields.
	//
	s.irccon = irc.IRC(u.User.Username(), u.User.Username())

	//
	// Do we have a password?  If so set it.
	//
	pass, passPresent := u.User.Password()
	if passPresent && pass != "" {
		s.irccon.Password = pass
	}

	s.irccon.Debug = false

	//
	// We assum "irc://...." by default, but if ircs:// was
	// specified we'll allow TLS.
	//
	s.irccon.UseTLS = false
	if u.Scheme == "ircs" {
		s.irccon.UseTLS = true
		s.irccon.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	//
	// Add a callback to join the channel
	//
	s.irccon.AddCallback("001", func(e *irc.Event) {
		s.channel = "#" + u.Fragment
		s.irccon.Join(s.channel)

		// Now we've joined
		s.joined = true
	})

	//
	// Because our connection is persistent we can use
	// it to process private messages.
	//
	// In this case we'll just say "No".
	//
	s.irccon.AddCallback("PRIVMSG", func(event *irc.Event) {
		go func(event *irc.Event) {
			//
			// event.Message() contains the message
			// event.Nick Contains the sender
			// event.Arguments[0] Contains the channel
			//
			// Send a private-reply.
			//
			s.mutex.Lock()
			var p = s.passed
			var f = s.failed
			s.mutex.Unlock()

			s.irccon.Privmsg(event.Nick,
				fmt.Sprintf("Total tests executed %d, %d passed, %d failed", p+f, p, f))
		}(event)
	})

	//
	// Connect
	//
	err = s.irccon.Connect(u.Host)
	if err != nil {
		return err
	}

	return nil
}

// Notify is the API-method which is invoked to send a notification
// somewhere.
//
// In our case we send a notification to the IRC server.
func (s *IRCNotifier) Notify(test test.Test, result error) error {

	//
	// If we don't have a server configured then return without sending
	// anything - there's no alternative since we don't know
	// which server/channel to use.
	//
	if s.data == "" {
		return nil
	}

	//
	// Bump our pass/fail counts.
	//
	if result == nil {
		s.mutex.Lock()
		s.passed += 1
		s.mutex.Unlock()
	} else {
		s.mutex.Lock()
		s.failed += 1
		s.mutex.Unlock()
	}

	//
	// If the test passed then we don't care.
	//
	if result == nil {
		return nil
	}

	//
	// Format the failure message.
	//
	msg := fmt.Sprintf("The %s test against %s failed: %s", test.Type, test.Target, result.Error())

	//
	// And send it.
	//
	if s.joined {
		s.irccon.Privmsg(s.channel, msg)
	} else {
		fmt.Printf("Sending message before IRC server joined!\n")
		return errors.New("Sending message before IRC server joined!")
	}

	return nil
}

// init is invoked to register our notifier at run-time.
func init() {
	Register("irc", func(data string) Notifier {
		return &IRCNotifier{data: data, joined: false}
	})
}
