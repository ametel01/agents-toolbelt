package pkgmgr

import (
	"slices"
	"sort"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
)

// DependencyPlanItem describes one missing install dependency that can be bootstrapped.
type DependencyPlanItem struct {
	Name       string
	Manager    Manager
	Method     catalog.InstallMethod
	RequiredBy []catalog.Tool
}

var bootstrapMethods = map[string]map[string]catalog.InstallMethod{
	"cargo": {
		"apt": {
			Manager:        "apt",
			Package:        "cargo",
			Command:        []string{"sudo", "apt", "install", "-y", "cargo"},
			TimeoutSeconds: 900,
		},
		"brew": {
			Manager:        "brew",
			Package:        "rust",
			Command:        []string{"brew", "install", "rust"},
			TimeoutSeconds: 900,
		},
		"dnf": {
			Manager:        "dnf",
			Package:        "cargo",
			Command:        []string{"sudo", "dnf", "install", "-y", "cargo"},
			TimeoutSeconds: 900,
		},
		"pacman": {
			Manager:        "pacman",
			Package:        "rust",
			Command:        []string{"sudo", "pacman", "-S", "--noconfirm", "rust"},
			TimeoutSeconds: 900,
		},
	},
	"go": {
		"apt": {
			Manager:        "apt",
			Package:        "golang-go",
			Command:        []string{"sudo", "apt", "install", "-y", "golang-go"},
			TimeoutSeconds: 900,
		},
		"brew": {
			Manager:        "brew",
			Package:        "go",
			Command:        []string{"brew", "install", "go"},
			TimeoutSeconds: 900,
		},
		"dnf": {
			Manager:        "dnf",
			Package:        "golang",
			Command:        []string{"sudo", "dnf", "install", "-y", "golang"},
			TimeoutSeconds: 900,
		},
		"pacman": {
			Manager:        "pacman",
			Package:        "go",
			Command:        []string{"sudo", "pacman", "-S", "--noconfirm", "go"},
			TimeoutSeconds: 900,
		},
	},
	"pipx": {
		"apt": {
			Manager:        "apt",
			Package:        "pipx",
			Command:        []string{"sudo", "apt", "install", "-y", "pipx"},
			TimeoutSeconds: 900,
		},
		"brew": {
			Manager:        "brew",
			Package:        "pipx",
			Command:        []string{"brew", "install", "pipx"},
			TimeoutSeconds: 900,
		},
		"dnf": {
			Manager:        "dnf",
			Package:        "pipx",
			Command:        []string{"sudo", "dnf", "install", "-y", "pipx"},
			TimeoutSeconds: 900,
		},
		"pacman": {
			Manager:        "pacman",
			Package:        "python-pipx",
			Command:        []string{"sudo", "pacman", "-S", "--noconfirm", "python-pipx"},
			TimeoutSeconds: 900,
		},
	},
}

// IsSecondaryManager reports whether a manager is a tool-specific dependency rather than a system package manager.
func IsSecondaryManager(manager string) bool {
	switch manager {
	case "cargo", "go", "pipx":
		return true
	default:
		return false
	}
}

// ResolveDependencies returns missing install-manager dependencies that can be bootstrapped
// through currently available package managers.
func ResolveDependencies(selected []catalog.Tool, available []Manager) []DependencyPlanItem {
	if len(selected) == 0 || len(available) == 0 {
		return nil
	}

	dependencies := make(map[string]*DependencyPlanItem)
	for _, tool := range selected {
		if _, _, err := SelectBest(tool, available); err == nil {
			continue
		}

		dependency, ok := resolveDependencyForTool(tool, available)
		if !ok {
			continue
		}

		existing := dependencies[dependency.Name]
		if existing == nil {
			dependency.RequiredBy = []catalog.Tool{tool}
			dependencies[dependency.Name] = &dependency

			continue
		}

		existing.RequiredBy = append(existing.RequiredBy, tool)
	}

	resolved := make([]DependencyPlanItem, 0, len(dependencies))
	for _, managerName := range managerPriority {
		dependency, ok := dependencies[managerName]
		if !ok {
			continue
		}

		sort.SliceStable(dependency.RequiredBy, func(left, right int) bool {
			return dependency.RequiredBy[left].Name < dependency.RequiredBy[right].Name
		})
		resolved = append(resolved, *dependency)
	}

	return resolved
}

func resolveDependencyForTool(tool catalog.Tool, available []Manager) (DependencyPlanItem, bool) {
	methodManagers := make(map[string]struct{}, len(tool.InstallMethods))
	for _, method := range tool.InstallMethods {
		methodManagers[method.Manager] = struct{}{}
	}

	for _, managerName := range managerPriority {
		if _, ok := methodManagers[managerName]; !ok {
			continue
		}

		method, manager, ok := bootstrapMethod(managerName, available)
		if !ok {
			continue
		}

		return DependencyPlanItem{
			Name:    managerName,
			Manager: manager,
			Method:  method,
		}, true
	}

	return DependencyPlanItem{}, false
}

func bootstrapMethod(name string, available []Manager) (catalog.InstallMethod, Manager, bool) {
	methods, ok := bootstrapMethods[name]
	if !ok {
		return catalog.InstallMethod{}, nil, false
	}

	for _, managerName := range managerPriority {
		method, methodOK := methods[managerName]
		if !methodOK {
			continue
		}

		index := slices.IndexFunc(available, func(manager Manager) bool {
			return manager.Name() == managerName
		})
		if index < 0 {
			continue
		}

		return method, available[index], true
	}

	return catalog.InstallMethod{}, nil, false
}
