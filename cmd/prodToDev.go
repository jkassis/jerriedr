package main

// conf for the dev service
var prodToDevServiceSpecs []string = []string{
	"local|dockie|10001|/v1/Backup|/v1/Restore/Dockie",
	"local|dubbie|10001|/v1/Backup|/v1/Restore/Other",
	"local|keevie|10001|/v1/Backup|/v1/Restore/Other",
	"local|ledgie|10001|/v1/Backup|/v1/Restore/Other",
	"local|permie|10001|/v1/Backup|/v1/Restore/Other",
	"local|tickie|10001|/v1/Backup|/v1/Restore/Other",
}

var prodToDevArchiveSpecs []string = []string{
	"local|multi|/var/multi/single/local-server-0",
	"local|dubbie|/var/multi/single/local-server-0",
	"local|keevie|/var/multi/single/local-server-0",
	"local|ledgie|/var/multi/single/local-server-0",
	"local|permie|/var/multi/single/local-server-0",
	"local|tickie|/var/multi/single/local-server-0",
}

var prodToDevRepoArchiveSpecs []string = prodRepoArchiveSpecs
