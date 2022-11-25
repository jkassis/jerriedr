package main

// conf for the dev service
var devFromProdServiceSpecs []string = []string{
	"local|dockie|10001|/v1/Backup|/v1/Restore/Dockie",
	"local|dubbie|10001|/v1/Backup|/v1/Restore/Other",
	"local|keevie|10001|/v1/Backup|/v1/Restore/Other",
	"local|ledgie|10001|/v1/Backup|/v1/Restore/Other",
	"local|permie|10001|/v1/Backup|/v1/Restore/Other",
	"local|tickie|10001|/v1/Backup|/v1/Restore/Other",
}

var devFromProdArchiveSpecs []string = []string{
	"local|multi|/var/multi/single/local-server-0",
	"local|dubbie|/var/multi/single/local-server-0",
	"local|keevie|/var/multi/single/local-server-0",
	"local|ledgie|/var/multi/single/local-server-0",
	"local|permie|/var/multi/single/local-server-0",
	"local|tickie|/var/multi/single/local-server-0",
}

var devServiceSpecs []string = []string{
	"local|multi|10001|/v1/Backup|/v1/Restore/Dockie",
}

var devArchiveSpecs []string = []string{
	"local|multi|/var/multi/single/local-server-0",
}

// conf for local storage of devArchive
var localDevArchiveSpecs []string = []string{
	"local|multi|/var/jerrie/archive/dev/multi",
}
