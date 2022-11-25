package main

// conf for the dev service
var devServiceSpecs []string = []string{
	"local|10001",
}
var devArchiveSpecs []string = []string{
	"statefulset|fg/dockie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/ledgie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/tickie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/dubbie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/keevie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/permie|/var/data/single/<pod>-server-0/backup",
}
var devServiceRestoreFolder string = "/var/multi/single/local-server-0/restore"

// conf for local storage of devArchive
var localDevArchiveSpec string = "local|/var/jerrie/archive/dev"
var localDevServiceArchiveSpecs []string = []string{
	"local|/var/jerrie/archive/dev/multi",
}
