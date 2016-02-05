package main

import(
	"log"
	"flag"
	"os"
	"net/rpc"
	"os/exec"
	"io/ioutil"
	"runtime"
	"sync"
	"fmt"
	"bitbucket.org/racam/make-go/deptree"
	"bitbucket.org/racam/make-go/job"
)

//Quand on recoit un tâche:
//On recoit les fichiers nécessaire et les commandes depuis le master
//On execute la commande
//On regarde si un fichier portant le nom de la rèlge existe
//Si oui on le renvoi au serveur
//Sinon Erreur

func main() {
	
	var serverAddress string
	var displayHelp bool
	var nbThread int

	flag.BoolVar(&displayHelp, "help", false, "Display help")
	flag.StringVar(&serverAddress, "server", "localhost:9876", "address of the server with port")
	flag.IntVar(&nbThread, "thread", 1, "Number of thread to execute")
	flag.Parse()

	if displayHelp {
		flag.PrintDefaults()
		return
	}

	//Connection au serveur
	log.Println("Connecting to the serveur :",serverAddress)
	client, err := rpc.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	log.Println("Connection established")
	defer client.Close()

	//On borne le nombre de goroutine par le nombre de coeur dispo
	maxThread := maxParallelism()
	if nbThread > maxThread {
		nbThread = maxThread
	}

	
	//On lance les goroutines
	var threadWait sync.WaitGroup
	for i := 0; i < nbThread; i++ {
		threadWait.Add(1)
		go thread(client, &threadWait)
	}

	threadWait.Wait()
}

func thread(client *rpc.Client, w *sync.WaitGroup) {
	defer w.Done()
	
	//Télécharge et traite des jobs
	for {

		//Récupère le prochain job
		log.Println("Call for a new job")
		var t deptree.Node
		err := client.Call("Job.StartJob", 0, &t)
		
		//No more job
		if err != nil {
			log.Println("Server Job.StartJob No more job:", err)
			return
		}

		log.Println("New job download:", t)

		//Récupère les fichiers manquant
		getMissingFiles(&t, client)

		//On exécute les commandes
		out := doCmds(&t)

		//On prépare la réponse
		f, err := job.GetFile(t.Target)
		
		if os.IsNotExist(err) {
			log.Println("No reply file found")
		}

		reply := job.SlaveResponse{t.Target, f, out}

		//On Délivre le résultat
		log.Println("Start uploading reply :", reply.Target)
		var a int
		err = client.Call("Job.FinishJob", reply, &a)
		if err != nil {
			log.Println("Server Job.FinishJob:", err)
			return
		}
		log.Println("Upload Done")
	}
}

func getMissingFiles(n *deptree.Node, client *rpc.Client) {
	
	log.Println("Looking for some Missing Files")
	var w sync.WaitGroup

	for i, filename := range n.Deps {
		
		//Si le fichier n'existe pas localement
		if _, err := os.Stat(filename); os.IsNotExist(err) {		

			log.Printf("[%v]%v needed", i, filename)
			w.Add(1)
			go func(tmp string) {
				defer w.Done()

				//On télécharge le fichier
				var f job.File
				err = client.Call("Job.GetFile", tmp, &f)
				if err != nil {
					log.Println("file:", tmp, " not found - Should be a intermediate target")
				} else {

					//On le copie localement
					err = ioutil.WriteFile(f.Name, f.Content, f.Mode)
					if err != nil {
						log.Fatal(err)
					}
					
					log.Println("Download file:", tmp, " OK!")
				}
			}(filename)
		}
	}

	w.Wait()
}


func doCmds(n *deptree.Node) (res []string) {
	log.Println("Start executing commands")	
	for i, cmd := range n.Cmds {
		
		log.Printf("[%v] -- %v", i, cmd)
		out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
		res = append(res, fmt.Sprintf("%s\n%s",cmd, out))
		
		if err != nil {
			log.Fatal("Makefile: recipe for target '", n.Target,"' failed")
		}
	}
	return
}


// source:
// stackoverflow.com/questions/13234749/golang-how-to-verify-number-of-processors-on-which-a-go-program-is-running
func maxParallelism() int {
    maxProcs := runtime.GOMAXPROCS(0)
    numCPU := runtime.NumCPU()
    if maxProcs < numCPU {
        return maxProcs
    }
    return numCPU
}