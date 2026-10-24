package deployment

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeDeployer struct {
	err      error
	got      string
	size     int64
	filename string
	calls    int
}

func (f *fakeDeployer) DeployData(data io.Reader, size int64, filename string) error {
	body, _ := io.ReadAll(data)
	f.got = string(body)
	f.size = size
	f.filename = filename
	f.calls++
	return f.err
}

func TestMultiDeployerDeployData(t *testing.T) {
	t.Run("EveryDeployerGetsFullData", func(t *testing.T) {
		first, second := &fakeDeployer{}, &fakeDeployer{}
		multi := NewMultiDeployer(first, second)

		data := `{"faction":"test"}`
		if err := multi.DeployData(strings.NewReader(data), int64(len(data)), "travel_data.json"); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		for i, d := range []*fakeDeployer{first, second} {
			if d.got != data {
				t.Errorf("Deployer %d: expected data '%s', got '%s'", i, data, d.got)
			}
			if d.size != int64(len(data)) {
				t.Errorf("Deployer %d: expected size %d, got %d", i, len(data), d.size)
			}
			if d.filename != "travel_data.json" {
				t.Errorf("Deployer %d: expected filename 'travel_data.json', got '%s'", i, d.filename)
			}
		}
	})

	t.Run("FailureDoesNotStopOthers", func(t *testing.T) {
		errFirst := errors.New("ssh down")
		first, second := &fakeDeployer{err: errFirst}, &fakeDeployer{}
		multi := NewMultiDeployer(first, second)

		err := multi.DeployData(strings.NewReader("{}"), 2, "travel_data.json")
		if !errors.Is(err, errFirst) {
			t.Errorf("Expected error to wrap '%v', got %v", errFirst, err)
		}
		if second.calls != 1 {
			t.Errorf("Expected second deployer to be called once, got %d", second.calls)
		}
	})

	t.Run("ReturnsAllFailures", func(t *testing.T) {
		errFirst, errSecond := errors.New("ssh down"), errors.New("r2 down")
		multi := NewMultiDeployer(&fakeDeployer{err: errFirst}, &fakeDeployer{err: errSecond})

		err := multi.DeployData(strings.NewReader("{}"), 2, "travel_data.json")
		if !errors.Is(err, errFirst) || !errors.Is(err, errSecond) {
			t.Errorf("Expected error to wrap both failures, got %v", err)
		}
	})
}
