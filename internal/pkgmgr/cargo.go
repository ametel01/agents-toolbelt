package pkgmgr

func newCargoManager(lookPath lookPathFunc) Manager {
	return newCommandManager("cargo", lookPath)
}
