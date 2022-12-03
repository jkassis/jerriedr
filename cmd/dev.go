package main

var devServiceSpecs []string = []string{
	"local|multi|10001|/v1/Backup|/v1/Restore/Dockie",
}

var devArchiveSpecs []string = []string{
	"local|multi|/var/multi/single/local-server-0",
}

// conf for local storage of devArchive
var devBackupArchiveSpecs []string = []string{
	"local|multi|/var/jerrie/archive/dev/multi",
}
