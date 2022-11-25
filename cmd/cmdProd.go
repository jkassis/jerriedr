package main

// conf for prod service
var prodServiceSpecs []string = []string{
	"statefulset|fg/dockie|10000",
	"statefulset|fg/tickie|10000",
	"statefulset|fg/ledgie|10000",
	"statefulset|fg/dubbie|10000",
	"statefulset|fg/keevie|10000",
	"statefulset|fg/permie|10000",
}
var prodArchiveSpecs []string = []string{
	"statefulset|fg/dockie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/ledgie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/tickie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/dubbie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/keevie|/var/data/single/<pod>-server-0/backup",
	"statefulset|fg/permie|/var/data/single/<pod>-server-0/backup",
}

// conf for local storage for prodarchives
var localProdArchiveSpec string = "local|/var/jerrie/archive/prod"
var localProdServiceArchiveSpecs []string = []string{
	"local|/var/jerrie/archive/prod/dockie",
	"local|/var/jerrie/archive/prod/dubbie",
	"local|/var/jerrie/archive/prod/keevie",
	"local|/var/jerrie/archive/prod/ledgie",
	"local|/var/jerrie/archive/prod/permie",
	"local|/var/jerrie/archive/prod/tickie",
}
