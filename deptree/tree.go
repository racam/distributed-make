package deptree


type Node struct {
	Target		string 		//Nom de la règle
	Deps		[]string 	//Listes des dépendances
	Cmds		[]string 	//Liste de commandes à exécuter
	Sons		[]*Node 	//sous-ensemble de "deps" qui correspond à d'autres règles
	Affected	bool 		//règle affectée à un slave
	Done		bool 		//règle faite
}


func NewNode(target string, deps []string, cmds []string) *Node {
	return &Node{target, deps, cmds, nil, false, false}
}

func (n Node) String() string {
    return n.Target
}
