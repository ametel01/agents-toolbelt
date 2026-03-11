package pkgmgr

func newDNFManager(lookPath lookPathFunc) Manager {
	return newCommandManager("dnf", lookPath)
}
