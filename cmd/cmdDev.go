package main

var localDevServiceSpec string = "local|10001"
var localDevRestoreFolder string = "/var/multi/single/local-server-0/restore"

// used to get get snapshots from prod
// will contain one subfolder per stateful set service
var localProdArchiveSpec string = "local|/var/jerrie/archive/prod"

// used to put snapshots to prod and restore to dev
var localProdArchiveSpecs []string = []string{
	"local|/var/jerrie/archive/prod/dockie",
	"local|/var/jerrie/archive/prod/dubbie",
	"local|/var/jerrie/archive/prod/keevie",
	"local|/var/jerrie/archive/prod/ledgie",
	"local|/var/jerrie/archive/prod/permie",
	"local|/var/jerrie/archive/prod/tickie",
}
