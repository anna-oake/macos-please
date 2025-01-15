package main

import "time"

var Version = "(dev)" // set at compile time

type args struct {
	OutputDir   string        `arg:"-o,--output-dir,required" help:"output directory"`
	OnlyLatest  bool          `arg:"-l,--latest" default:"true" help:"only download the latest installer for each major version"`
	MajorLimit  int           `arg:"-m,--major-limit" default:"3" help:"limit the number of major versions to download (0 for no limit)"`
	KeepOld     bool          `arg:"-k,--keep-old" default:"false" help:"keep installers that don't meet the criteria anymore"`
	MistTimeout time.Duration `arg:"--mist-timeout" default:"1h" help:"timeout for Mist installer downloads"`
	MistCache   bool          `arg:"--mist-cache" default:"false" help:"cache Mist installer downloads"`
}

func (args) Version() string {
	return "macos-please " + Version
}
