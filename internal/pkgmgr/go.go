package pkgmgr

func newGoManager(lookPath lookPathFunc) Manager {
	return newCommandManager("go", lookPath)
}
