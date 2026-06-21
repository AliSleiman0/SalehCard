// Package transform applies SalehCard's migration business rules, turning legacy
// products/categories into the target seed schema (with review flags).
package transform

// RootIDs is the canonical set of legacy root categories to walk (spec §2).
var RootIDs = []int{61, 63, 65, 216, 235, 107, 115, 447}

// rootDomains maps each root category id to the rootDomain emitted on seeds.
var rootDomains = map[int]string{
	61:  "app_topups",
	63:  "games",
	65:  "telecom",
	216: "gsm_tools",
	235: "software",
	107: "wallets_crypto", // 107 is mislabeled "Cryptocurrency" — see run-summary note
	115: "giftcards",
	447: "money_transfers",
}

// Tree resolves a product's root domain by walking category parent links.
type Tree struct {
	parent map[int]int // childID -> parentID (roots absent / 0)
}

// NewTree builds a Tree from (childID, parentID) edges discovered during the walk.
func NewTree() *Tree { return &Tree{parent: map[int]int{}} }

// AddEdge records that childID's parent is parentID.
func (t *Tree) AddEdge(childID, parentID int) {
	if childID != 0 && parentID != 0 {
		t.parent[childID] = parentID
	}
}

// Root walks categoryID up the parent chain to one of the 8 roots, returning the
// root id and its domain. ok is false when the chain doesn't reach a known root.
func (t *Tree) Root(categoryID int) (rootID int, domain string, ok bool) {
	id := categoryID
	for i := 0; i < 64; i++ { // bounded against cycles
		if d, isRoot := rootDomains[id]; isRoot {
			return id, d, true
		}
		p, has := t.parent[id]
		if !has || p == 0 || p == id {
			return 0, "", false
		}
		id = p
	}
	return 0, "", false
}

// RootDomain returns the domain for a known root id.
func RootDomain(id int) (string, bool) { d, ok := rootDomains[id]; return d, ok }

// Depth returns how many hops categoryID is below its root (0 = root itself).
// Returns -1 when the chain doesn't reach a known root.
func (t *Tree) Depth(categoryID int) int {
	id := categoryID
	for d := 0; d < 64; d++ {
		if _, isRoot := rootDomains[id]; isRoot {
			return d
		}
		p, has := t.parent[id]
		if !has || p == 0 || p == id {
			return -1
		}
		id = p
	}
	return -1
}
