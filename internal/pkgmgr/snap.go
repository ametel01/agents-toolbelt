package pkgmgr

func newSnapManager(lookPath lookPathFunc) Manager {
	return newCommandManager("snap", lookPath)
}
