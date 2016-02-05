package parsing

import (
	"log"
	"os"
	"bufio"
	"errors"
	"strings"
	"bitbucket.org/racam/make-go/deptree"
)



//Etape 1 on parse les lignes
//pour chaque ligne, soit une regle soit une commande. Donc on crée un nouveau Node ou on ajoute la cmds au Node existant.

//Etape 2 on parcours toutes les deps de nos Nodes
//Si dep = Node existant -> créer une arête
//Sinon si un fichier porte ce nom -> Ok !
//Sinon erreur !

func Makefile(filename, target string) map[string]*deptree.Node {
	
	//Ouverture/fermeture du fichier
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	//Stockage des targets dans une map -> tab[target] = *Node
	listeRules := make(map[string]*deptree.Node)



	//ETAPE 1 : Parcours de chaque ligne du makefile
	scan := bufio.NewScanner(file)
	var currentNode *deptree.Node = nil

	for scan.Scan() {
		currentLine := scan.Text()

		//Si ligne vide ou commentaire on saute la ligne
		if strings.HasPrefix(currentLine, "#") || len(currentLine) == 0 {
			continue
		}

		//Si c'est une commande
		if strings.HasPrefix(currentLine, "\t") {
			//Pas de node
			if currentNode == nil {
				err = errors.New("Parsing Makefile : target must be set before the command line")
				log.Fatal(err)
			}

			//On ajoute la commande au node courant
			currentNode.Cmds = append(currentNode.Cmds, strings.TrimSpace(currentLine))
		
		//Si c'est une nouvelle cible
		} else {
			
			if !strings.Contains(currentLine, ":") {
				log.Fatal("Parsing Makefile: can't find separator ':' in line : ", currentLine)
			}

			//On découpe la ligne en "targetTmp : deps1 deps2"
			line := strings.SplitN(currentLine, ":", 2)
			
			targetTmp := strings.TrimSpace(line[0])
			deps := strings.Fields(line[1])
			
			currentNode = deptree.NewNode(targetTmp, deps, nil)
			listeRules[targetTmp] = currentNode
		}
	}

	//On regarde si la règle que l'on veut exécuter existe
	if _, ok := listeRules[target]; !ok {
		log.Fatal("Parsing Makefile: Target not found: ", target)
	}


	//ETAPE 2 on crée les liaisons entre node pour matérialiser les dépendances
	for _, node := range listeRules {

		//log.Print(node.Deps)
		for _, dep := range node.Deps {


			//On cherche si un fichier existe pour cette dépendance
			if _, err := os.Stat(dep); os.IsNotExist(err) {
				
				//Si la dépendance est une règle existante on l'ajoute aux fils
				if son, ok := listeRules[dep]; ok {
					node.Sons = append(node.Sons, son)
					continue
				}

				log.Fatal("No rule to make target '", dep, "', needed by '", node.Target,"'")	
			}
		}
	}

	return listeRules
}