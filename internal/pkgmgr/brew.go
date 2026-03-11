package pkgmgr

func newBrewManager(lookPath lookPathFunc) Manager {
	return newCommandManager("brew", lookPath)
}
