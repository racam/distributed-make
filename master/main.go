package main

import (
	"os"
	"log"
	"net"
	"flag"
	"net/rpc"
	"bitbucket.org/racam/make-go/parsing"
	"bitbucket.org/racam/make-go/job"
	"bitbucket.org/racam/make-go/deptree"
)

//Pour envoyer une tâche à un slave:
//On vérifie que tous les Nodes fils ont été fait
//On parcours les deps de cette tâche et on regarde lesquelles sont des fichiers
//On transfert tous ces fichiers au slave
//le slave peut executer la tache

func main() {
	
	var pathMakefile, target, address string
	var displayHelp bool

	flag.BoolVar(&displayHelp, "help", false, "Display help")
	flag.StringVar(&pathMakefile, "makefile", "Makefile", "Path to the Makefile")
	flag.StringVar(&target, "target", "out.avi", "Target to execute")
	flag.StringVar(&address, "address", "localhost:9876", "address to listen")
	flag.Parse()

	
	if displayHelp {
		flag.PrintDefaults()
		return
	}

	//Test si le fichier Makefile existe bien
	if _, err := os.Stat(pathMakefile); os.IsNotExist(err) {
		log.Fatal("Makefile not found :", pathMakefile)
	}

	//Parse le makefile et récupère la règle root
	log.Println("Makefile parsing Start")
	listRules := parsing.Makefile(pathMakefile, target)
	log.Println("Makefile parsing Done")
	

	//On initialise nos jobs qui vont distribuer les règles aux slaves
	j := job.NewJob(listRules, target)

	
	//On lance la recherche des règles éligibles en background
	go findTargets(listRules[target], j)


	//Setup du serveur
	log.Println("Start server on address:", address)
	
	rpc.Register(j)
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	go rpc.Accept(l)

	log.Println("Ready to accept new connection")
	

	//Attend le signal que toutes les targets soient faites
	<- j.Done

	l.Close()
	log.Println("Server shuting down")
}


func findTargets(n *deptree.Node, j *job.Job) {

	//Tant que la target principale n'est pas affectée
	for {
		
		log.Println("Start finding next target")
		j.UpdateNode.RLock()
		res := nextTarget(n)
		j.UpdateNode.RUnlock()


		//On envoie la prochaine target au Job
		if res != nil {
			log.Println("Next target found:", res.Target)
			j.NextJob <- res.Target
			
			//Si on a délivré le dernier job on break
			if res.Target == j.MainTarget {
				break
			}

		} else{
			j.Wait = true
			log.Println("No new target found, now wait")
			<- j.WakeUp
			j.Wait = false
			log.Println("wakeup")
		}
	}

	//Notre target principale est affectée : no more job
	log.Println("Stop finding next target")
	close(j.NextJob)
}


func nextTarget(n *deptree.Node) *deptree.Node {
	//Si pas de fils GO !
	if n.Sons == nil {
		return n
	}

	AllDone := true

	for _, son := range n.Sons {
		//Si un fils n'est pas done
		if !son.Done {
			AllDone = false
		}

		if !son.Affected && !son.Done {
			//Si on remonte un fils éligible
			if res := nextTarget(son); res != nil {
				return res
			}
		}
	}

	//tous les fils sont Done : GO !
	if AllDone {
		return n
	}

	//tous les fils sont Affected : invalide
	return nil
}
