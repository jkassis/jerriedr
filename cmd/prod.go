package main

// conf for prod service
var prodServiceSpecs []string = []string{
	"statefulset|fg/dockie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
	"statefulset|fg/tickie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
	"statefulset|fg/ledgie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
	"statefulset|fg/dubbie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
	"statefulset|fg/keevie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
	"statefulset|fg/permie|10000|/v1/Backup|/v1/Restore|/var/data/single/<pod>-server-0/restore",
}

var prodSnapArchiveSpecs []string = []string{
	"statefulset|fg/dockie|/var/data/single/<pod>-server-0",
	"statefulset|fg/ledgie|/var/data/single/<pod>-server-0",
	"statefulset|fg/tickie|/var/data/single/<pod>-server-0",
	"statefulset|fg/dubbie|/var/data/single/<pod>-server-0",
	"statefulset|fg/keevie|/var/data/single/<pod>-server-0",
	"statefulset|fg/permie|/var/data/single/<pod>-server-0",
}

// conf for local storage for prodArchive
var prodBackupArchiveSpecs []string = []string{
	"local|dockie|/var/jerrie/archive/prod/dockie",
	"local|dubbie|/var/jerrie/archive/prod/dubbie",
	"local|keevie|/var/jerrie/archive/prod/keevie",
	"local|ledgie|/var/jerrie/archive/prod/ledgie",
	"local|permie|/var/jerrie/archive/prod/permie",
	"local|tickie|/var/jerrie/archive/prod/tickie",
}
