// Based on:
// https://github.com/emersion/go-mbox/blob/master/reader.go
// https://github.com/tvanriper/mbox/blob/main/reader_test.go
package mbox

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"testing"
)

const mboxWithOneMessage = `From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.
`

const mboxWithThreeMessages = `From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.

From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: derp.herp@example.com (Derp Herp)
Date: Thu, 02 Jan 2015 00:00:01 +0100
Subject: Another test

This is another simple test.

Another line.

Bye.

From bernd.lauert@example.com Thu Jan  3 00:00:01 2015
From: bernd.lauert@example.com (Bernd Lauert)
Date: Thu, 03 Jan 2015 00:00:01 +0100
Subject: A last test

This is the last simple test.

Bye.
`

const mboxWithStartingLF = `
From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.

From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: derp.herp@example.com (Derp Herp)
Date: Thu, 02 Jan 2015 00:00:01 +0100
Subject: Another test

This is another simple test.

Another line.

Bye.

From bernd.lauert@example.com Thu Jan  3 00:00:01 2015
From: bernd.lauert@example.com (Bernd Lauert)
Date: Thu, 03 Jan 2015 00:00:01 +0100
Subject: A last test

This is the last simple test.

Bye.
`

const mboxWithThreeMessagesMalformedButValid = `From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.
From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: derp.herp@example.com (Derp Herp)
Date: Thu, 02 Jan 2015 00:00:01 +0100
Subject: Another test

This is another simple test.

Another line.

Bye.

From bernd.lauert@example.com Thu Jan  3 00:00:01 2015
From: bernd.lauert@example.com (Bernd Lauert)
Date: Thu, 03 Jan 2015 00:00:01 +0100
Subject: A last test

This is the last simple test.

Bye.
`

const mboxWithOneMessageMissingSeparator = `From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.
`

const mboxFirstMessage = `From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

And, by the way, this is how a "From" line is escaped in mboxo format:

>From Herp Derp with love.

Bye.
`

const mboxo = `From someone
From: bubbles@bubbletown.com
To: mrmxpdstk@lazytown.com
Subject: To interpretation

>From all of us, to all of you, be happy!
From someone-else
From: mrspam@corporate.corp.com
To: mrmxpdstk@lazytown.com
Subject: Bestest offer in the universe!!11!!

You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
`

const mboxcl = `From someone
From: bubbles@bubbletown.com
To: mrmxpdstk@lazytown.com
Subject: To interpretation
Content-Length: 42

>From all of us, to all of you, be happy!
From someone-else
>From mug: weird header
Content-Length: 130
From: mrspam@corporate.corp.com
To: mrmxpdstk@lazytown.com
Subject: Bestest offer in the universe!!11!!

You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
From nobody
From: nobody@nowhere.man
To: mrmxpdstk@lazytown.com
Subject: Mysterious Jenkins
Content-Length: 0

`

const badmboxcl = `From someone
From: bubbles@bubbletown.com
To: mrmxpdstk@lazytown.com
Subject: To interpretation
Content-Length: ts

>From all of us, to all of you, be happy!
From someone-else
Content-Length: 130
From: mrspam@corporate.corp.com
To: mrmxpdstk@lazytown.com
Subject: Bestest offer in the universe!!11!!

You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
`

func crlfToLf(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func testMboxMessage(t *testing.T, mbox string, count int) {
	b := bytes.NewBufferString(mbox)
	m := NewReader(b)

	for i := 0; i < count; i++ {
		r, err := m.NextMessage()
		if err != nil {
			t.Fatalf("Unexpected error after NextMessage(): %v", err)
		}

		var text bytes.Buffer
		_, err = text.ReadFrom(r)
		if err != nil {
			t.Errorf("Unexpected error reading message body: %v", err)
		}
		want := crlfToLf(mboxFirstMessage)
		tmp := crlfToLf(text.String())
		if i == 0 && tmp != want {
			t.Errorf("Expected:\n %q\ngot\n%q", want, tmp)
		}
	}

	if _, err := m.NextMessage(); err != io.EOF {
		t.Fatalf("Unexpected error after NextMessage(): %v", err)
	}
}

func TestMboxMessageWithOneMessage(t *testing.T) {
	testMboxMessage(t, mboxWithOneMessage, 1)
}

func TestMboxMessageWithThreeMessages(t *testing.T) {
	testMboxMessage(t, mboxWithThreeMessages, 3)
}

func TestMboxMessageWithStartingLF(t *testing.T) {
	testMboxMessage(t, mboxWithStartingLF, 3)
}

func TestMboxMessageWithThreeMessagesMalformedButValid(t *testing.T) {
	testMboxMessage(t, mboxWithThreeMessagesMalformedButValid, 3)
}

func testMboxMessageInvalid(t *testing.T, mbox string) {
	b := bytes.NewBufferString(mbox)
	m := NewReader(b)

	if _, err := m.NextMessage(); err == nil {
		t.Errorf("Missing error after Next(): %v", err)
	}
}

func TestMboxMessageWithOneMessageMissingSeparator(t *testing.T) {
	testMboxMessageInvalid(t, mboxWithOneMessageMissingSeparator)
}

func TestMboxMessageNoRead(t *testing.T) {
	b := bytes.NewBufferString(mboxWithThreeMessages)
	m := NewReader(b)

	n := 0
	for {
		_, err := m.NextMessage()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("Unexpected error after NextMessage(): %v", err)
		}
		n++
	}

	if _, err := m.NextMessage(); err != io.EOF {
		t.Fatalf("Unexpected error after NextMessage(): %v", err)
	}
	if n != 3 {
		t.Fatalf("Expected 3 mesages, got %v", n)
	}
}

// TODO: decide whether we should keep this test or not
func DisabledTestScanMessageWithBoundaries(t *testing.T) {
	sourceData := `
From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test
Content-Type: multipart/alternative;
        boundary=Apple-Mail-D55D9B1A-A379-4D5C-BDA9-00D35DF424A0

This is a test of boundaries.  Don't accept a new email via \nFrom until the boundary is done!'

And, by the way, this is how a "From" line is escaped in mboxo format:
From Herp Derp with love.

From Herp Derp with love.

Bye.
--Apple-Mail-D55D9B1A-A379-4D5C-BDA9-00D35DF424A0--

From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is the second email in a test of boundaries.
`
	expected := []string{
		"This is a test of boundaries.  Don't accept a new email via \\nFrom until the boundary is done!'\n\nAnd, by the way, this is how a \"From\" line is escaped in mboxo format:\n>From Herp Derp with love.\n\nFrom Herp Derp with love.\n\nBye.\n--Apple-Mail-D55D9B1A-A379-4D5C-BDA9-00D35DF424A0--\n",
		"This is the second email in a test of boundaries.\n",
	}
	b := bytes.NewBufferString(sourceData)
	m := NewReader(b)

	for i := range expected {
		r, err := m.NextMessage()
		if err != nil {
			t.Fatalf("Unexpected error after NextMessage(): %v", err)
		}

		msg, err := mail.ReadMessage(r)
		if err != nil {
			t.Fatalf("mail.ReadMessage() = %v", err)
		}

		var body bytes.Buffer
		_, err = body.ReadFrom(msg.Body)
		if err != nil {
			t.Errorf("%d - Unexpected error reading message body: %v", i, err)
			continue
		}
		want := crlfToLf(expected[i])
		if body.String() != want {
			t.Errorf("%d - Expected:\n %q\ngot\n%q", i, want, body.String())
		}
	}

	if _, err := m.NextMessage(); err != io.EOF {
		t.Errorf("Next() succeeded")
	}
}

func TestScanMessageWithTextBoundary(t *testing.T) {
	sourceData := `
From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test
Content-Type: text/html; charset="utf-8";
 boundary="monkey_d3df4dc8-da5e-47dd-be15-f19c5ed55194"

This is a test of boundaries.  Don't accept a new email via \nFrom until the boundary is done!'

And, by the way, this is how a "From" line is escaped in mboxo format:

Bye.

From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is the second email in a test of boundaries.
`
	expected := []string{
		"This is a test of boundaries.  Don't accept a new email via \\nFrom until the boundary is done!'\n\nAnd, by the way, this is how a \"From\" line is escaped in mboxo format:\n\nBye.\n",
		"This is the second email in a test of boundaries.\n",
	}
	b := bytes.NewBufferString(sourceData)
	m := NewReader(b)

	for i := range expected {
		r, err := m.NextMessage()
		if err != nil {
			t.Fatalf("Unexpected error after NextMessage(): %v", err)
		}

		msg, err := mail.ReadMessage(r)
		if err != nil {
			t.Fatalf("mail.ReadMessage() = %v", err)
		}

		var body bytes.Buffer
		_, err = body.ReadFrom(msg.Body)
		if err != nil {
			t.Errorf("%d - Unexpected error reading message body: %v", i, err)
			continue
		}

		want := crlfToLf(expected[i])
		got := crlfToLf(body.String())
		if got != want {
			t.Errorf("%d - Expected:\n %q\ngot\n%q", i, want, got)
		}
	}

	if _, err := m.NextMessage(); err != io.EOF {
		t.Errorf("Next() succeeded")
	}
}

func TestScanMessageWithBoundarySemicolon(t *testing.T) {
	mbox := `From notifications@github.com Tue Jun  7 05:46:46 2016
From: Sender <notifications@github.com>
To: foo/bar <bar@noreply.github.com>
Message-ID: <foo/bar/1/0@github.com>
Subject: Re: [foo/bar] [question] Baz? (#1)
Content-Type: multipart/alternative; boundary="--==_mimepart_5755da228145a_38da3facdf97329c42987b";
MIME-Version: 1.0


----==_mimepart_5755da228145a_38da3facdf97329c42987b
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 8bit

Blah blah

----==_mimepart_5755da228145a_38da3facdf97329c42987b
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: 8bit

<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<p>Blah blah</p>

----==_mimepart_5755da228145a_38da3facdf97329c42987b--

From notifications@github.com Tue Jun  7 05:52:15 2016
From: Author <notifications@github.com>
To: frob/blab <blab@noreply.github.com>
Message-ID: <frob/blab/1/0@github.com>
Subject: Re: [frob/blab] [question] Bling? (#1)
Content-Type: multipart/alternative; boundary="--==_mimepart_5755db739a819_79783f996b0172c04025ee";
MIME-Version: 1.0


----==_mimepart_5755db739a819_79783f996b0172c04025ee
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: 8bit

Blah blah

----==_mimepart_5755db739a819_79783f996b0172c04025ee
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: 8bit

<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<p>Blah blah</p>

----==_mimepart_5755db739a819_79783f996b0172c04025ee--

`
	expected := 2

	b := bytes.NewBufferString(mbox)
	m := NewReader(b)

	n := 0
	for {
		_, err := m.NextMessage()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("m.NextMessaage() = %v", err)
		}
		n += 1
	}

	if n != expected {
		t.Errorf("Expected: %d; got: %d", expected, n)
	}
}

func TestReadingMessageWithLongLine(t *testing.T) {
	// We are testing longer line than what `bufio.MaxScanTokenSize` defines.
	// RFC 5322 section 2.1.1 says line MUST be less than 998 bytes long, but
	// some providers generates messages with such long lines.
	want := strings.Repeat("This is very long line.", 5000) // Over 100k bytes.
	mbox := fmt.Sprintf(`From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

%s`, want)

	b := bytes.NewBufferString(mbox)
	m := NewReader(b)

	r, err := m.NextMessage()
	if err != nil {
		t.Fatalf("m.NextMessaage() = %v", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Fatalf("mail.ReadMessage() = %v", err)
	}

	var body bytes.Buffer
	_, err = body.ReadFrom(msg.Body)
	if err != nil {
		t.Errorf("body.ReadFrom() = %v", err)
	}

	if body.String() != want+"\r\n" {
		t.Errorf("Expected:\n %q\ngot\n%q", want, body.String())
	}

	_, err = m.NextMessage()
	if err != io.EOF {
		t.Fatalf("m.NextMessage() = %v", err)
	}
}

func ExampleReader() {
	r := strings.NewReader(`From herp.derp@example.com Thu Jan  1 00:00:01 2015
From: herp.derp@example.com (Herp Derp)
Date: Thu, 01 Jan 2015 00:00:01 +0100
Subject: Test

This is a simple test.

CU.

From derp.herp@example.com Thu Jan  1 00:00:01 2015
From: derp.herp@example.com (Derp Herp)
Date: Thu, 02 Jan 2015 00:00:01 +0100
Subject: Another test

This is another simple test.

Bye.
`)

	mr := NewReader(r)
	for {
		r, err := mr.NextMessage()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Print("Oops, something went wrong!", err)
			return
		}

		msg, err := mail.ReadMessage(r)
		if err != nil {
			fmt.Print("Oops, something went wrong!", err)
			return
		}

		fmt.Printf("Message from %v\n", msg.Header.Get("From"))
	}

	// Output:
	// Message from herp.derp@example.com (Herp Derp)
	// Message from derp.herp@example.com (Derp Herp)
}

type MsgTest struct {
	Headers map[string]string
	Body    string
}

func CheckMessage(expected MsgTest, msg *mail.Message) (err error) {
	for header, value := range expected.Headers {
		found := msg.Header.Get(header)
		if found != value {
			return fmt.Errorf("expected %s but found %s", value, found)
		}
	}

	b, err := io.ReadAll(msg.Body)
	if err != nil {
		return err
	}

	want := strings.TrimSpace(crlfToLf(expected.Body))
	got := strings.TrimSpace(crlfToLf(string(b)))
	if got != want {
		return fmt.Errorf("body of email does not match expectations\nExpected:\n%q\n\nGot (len:%d):\n%q\n", want, len(b), got)
	}

	return err
}

func TestReadMBOXO(t *testing.T) {
	box := NewReader(bytes.NewBuffer([]byte(mboxo)))
	r, err := box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	if msg == nil {
		t.Fatalf("no message from mail.ReadMessage")
	}

	headers := map[string]string{}
	headers["Subject"] = "To interpretation"
	headers["From"] = "bubbles@bubbletown.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ">From all of us, to all of you, be happy!\n",
	}, msg)

	if err != nil {
		t.Error(err)
	}

	r, _ = box.NextMessage()
	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Errorf("ReadMessage failed: %s\n", err)
	}
	if msg == nil {
		t.Fatalf("no message from mail.ReadMessage")
	}

	headers = map[string]string{}
	headers["Subject"] = "Bestest offer in the universe!!11!!"
	headers["From"] = "mrspam@corporate.corp.com"
	headers["To"] = "mrmxpdstk@lazytown.com"

	msgTest := MsgTest{
		Headers: headers,
		Body: `You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!`,
	}

	err = CheckMessage(msgTest, msg)

	if err != nil {
		t.Error(err)
	}

	_, err = box.NextMessage()
	if err == nil {
		t.Errorf("expected error, but got nil")
	}

	if err != io.EOF {
		t.Errorf("expected io.EOF but got %s", err)
	}
}

func TestReadMBOXRC(t *testing.T) {
	box := NewReader(bytes.NewBuffer([]byte(mboxo)))

	r, err := box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers := map[string]string{}
	headers["Subject"] = "To interpretation"
	headers["From"] = "bubbles@bubbletown.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ">From all of us, to all of you, be happy!\n",
	}, msg)
	if err != nil {
		t.Error(err)
	}

	r, _ = box.NextMessage()
	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers = map[string]string{}
	headers["Subject"] = "Bestest offer in the universe!!11!!"
	headers["From"] = "mrspam@corporate.corp.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body: `You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
`,
	}, msg)

	if err != nil {
		t.Error(err)
	}

	_, err = box.NextMessage()
	if err == nil {
		t.Errorf("expected an error but got nil")
	}

	if err != io.EOF {
		t.Errorf("expected an io.EOF error but got %s", err)
	}
}

func TestReadMBOXCL(t *testing.T) {
	box := NewReader(bytes.NewBuffer([]byte(mboxcl)))

	r, err := box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers := map[string]string{}
	headers["Subject"] = "To interpretation"
	headers["From"] = "bubbles@bubbletown.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ">From all of us, to all of you, be happy!\n",
	}, msg)

	if err != nil {
		t.Error(err)
	}

	r, err = box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers = map[string]string{}
	headers["Subject"] = "Bestest offer in the universe!!11!!"
	headers["From"] = "mrspam@corporate.corp.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body: `You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
`,
	}, msg)

	if err != nil {
		t.Error(err)
	}

	r, _ = box.NextMessage()
	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers = map[string]string{}
	headers["Subject"] = "Mysterious Jenkins"
	headers["From"] = "nobody@nowhere.man"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ``,
	}, msg)

	if err != nil {
		t.Error(err)
	}

	_, err = box.NextMessage()
	if err == nil {
		t.Errorf("expected an error but got nil")
	}

	if err != io.EOF {
		t.Errorf("expected an io.EOF error but got %s", err)
	}
}

func TestReadMBOXCL2(t *testing.T) {
	box := NewReader(bytes.NewBuffer([]byte(mboxcl)))

	r, err := box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers := map[string]string{}
	headers["Subject"] = "To interpretation"
	headers["From"] = "bubbles@bubbletown.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ">From all of us, to all of you, be happy!\n",
	}, msg)
	if err != nil {
		t.Error(err)
	}

	r, err = box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers = map[string]string{}
	headers["Subject"] = "Bestest offer in the universe!!11!!"
	headers["From"] = "mrspam@corporate.corp.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body: `You won't believe these prices!
>From 1 cent to 11 cents, we carry the least expensive
line of jets this side of the Gobi Desert!
`,
	}, msg)

	if err != nil {
		t.Error(err)
	}

	r, _ = box.NextMessage()
	msg, err = mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers = map[string]string{}
	headers["Subject"] = "Mysterious Jenkins"
	headers["From"] = "nobody@nowhere.man"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    ``,
	}, msg)

	if err != nil {
		t.Error(err)
	}

	_, err = box.NextMessage()
	if err == nil {
		t.Errorf("expected an error but got nil")
	}

	if err != io.EOF {
		t.Errorf("expected an io.EOF error but got %s", err)
	}
}

func TestReadMBOXCLBadContentLength(t *testing.T) {
	box := NewReader(bytes.NewBuffer([]byte(badmboxcl)))
	r, err := box.NextMessage()
	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	msg, err := mail.ReadMessage(r)
	if err != nil {
		t.Error(err)
	}

	headers := map[string]string{}
	headers["Subject"] = "To interpretation"
	headers["From"] = "bubbles@bubbletown.com"
	headers["To"] = "mrmxpdstk@lazytown.com"
	err = CheckMessage(MsgTest{
		Headers: headers,
		Body:    `>From all of us, to all of you, be happy!`,
	}, msg)

	if err != nil {
		t.Error(err)
	}
}
