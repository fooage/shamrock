package filestore

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test_filestoreServer_bufferReadFile(t *testing.T) {
	file, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	// Write large amount of text data to file.
	text := "this is some text data.\n"
	for i := 0; i < 100000; i++ {
		if _, err := file.WriteString(text); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	// Test bufferReadFile function.
	ins := &filestoreServer{}
	result, err := ins.bufferReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	expected := make([]byte, 0, len(text)*100000)
	for i := 0; i < 100000; i++ {
		expected = append(expected, []byte(text)...)
	}
	if string(result) != string(expected) {
		t.Errorf("expected '%s', but got '%s'", string(expected), string(result))
	}
}

func Test_filestoreServer_bufferWriteFile(t *testing.T) {
	file, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	// Write large amount of text data to buffer.
	text := "this is some text data.\n"
	data := make([]byte, 0, len(text)*100000)
	for i := 0; i < 100000; i++ {
		data = append(data, []byte(text)...)
	}

	// Test bufferWriteFile function.
	ins := &filestoreServer{}
	if err := ins.bufferWriteFile(file, data); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	expected := make([]byte, 0, len(text)*100000)
	for i := 0; i < 100000; i++ {
		expected = append(expected, []byte(text)...)
	}
	if string(result) != string(expected) {
		t.Errorf("expected '%s', but got '%s'", string(expected), string(result))
	}
}
