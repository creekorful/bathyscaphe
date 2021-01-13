package test

import (
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/process_mock"
	"github.com/golang/mock/gomock"
	"reflect"
	"testing"
)

// SubscriberDef is use to test subscriber definition
type SubscriberDef struct {
	Queue    string
	Exchange string
}

// CheckProcessFeatures check process defined features
func CheckProcessFeatures(t *testing.T, p process.Process, wantFeatures []process.Feature) {
	if !reflect.DeepEqual(p.Features(), wantFeatures) {
		t.Errorf("Differents flags: %v %v", p.Features(), wantFeatures)
	}
}

// CheckProcessCustomFlags check process defined custom flags
func CheckProcessCustomFlags(t *testing.T, p process.Process, wantFlags []string) {
	var names []string
	for _, customFlag := range p.CustomFlags() {
		names = append(names, customFlag.Names()[0])
	}

	if !reflect.DeepEqual(names, wantFlags) {
		t.Errorf("Differents flags: %v %v", names, wantFlags)
	}
}

// CheckInitialize check process initialization phase
func CheckInitialize(t *testing.T, p process.Process, callback func(provider *process_mock.MockProviderMockRecorder)) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	providerMock := process_mock.NewMockProvider(mockCtrl)
	callback(providerMock.EXPECT())

	if err := p.Initialize(providerMock); err != nil {
		t.Errorf("Error while Initializing process: %s", err)
	}
}

// CheckProcessSubscribers check process defined subscribers
func CheckProcessSubscribers(t *testing.T, p process.Process, subscribers []SubscriberDef) {
	var defs []SubscriberDef
	for _, sub := range p.Subscribers() {
		defs = append(defs, SubscriberDef{
			Queue:    sub.Queue,
			Exchange: sub.Exchange,
		})
	}

	if !reflect.DeepEqual(defs, subscribers) {
		t.Errorf("Differents subscribers: %v %v", defs, subscribers)
	}
}

// TODO HTTPHandler
