package portal_test

type fakeRunner struct {
	outputs map[string]string
	runs    []string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}}
}

func (f *fakeRunner) Output(name string, args ...string) (string, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	return f.outputs[key], nil
}

func (f *fakeRunner) Run(name string, args ...string) error {
	key := name
	for _, a := range args {
		key += " " + a
	}
	f.runs = append(f.runs, key)
	return nil
}
