package job

import(
	"errors"
	"os"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"bitbucket.org/racam/make-go/deptree"
)

//Manage les jobs
type Job struct {
	MainTarget		string						//La target que l'utilisateur veut faire
	Rules			map[string]*deptree.Node 	//Clef=Rule --- Valeur=Node associé
	NextJob			chan string					//Communique un target éligible
	UpdateNode		sync.RWMutex				//Donne le signal de continuer la recherche de target
	Done			chan int					//Donne le signal que tous les Jobs sont Done
	WakeUp			chan int					//Donne le signal que l'on peut débloquer le wait
	Wait			bool
}


//Conteneur pour s'envoyer des fichiers
type File struct {
	Name		string
	Content		[]byte
	Mode		os.FileMode
}

//Résultat de l'exécution d'une target par un slave
type SlaveResponse struct {
	Target		string
	File		File
	Output		[]string
}


func NewJob(rules map[string]*deptree.Node, target string) *Job {
	j := new(Job)
	
	j.MainTarget		= target
	j.Rules				= rules
	j.Wait				= false
	j.NextJob			= make(chan string)
	j.Done				= make(chan int)
	j.WakeUp			= make(chan int)
	
	return j
}


func (j *Job) StartJob(a int, res *deptree.Node) error {
	log.Println("Slave wants to start a new job")
	
	//On récupère la prochaine target éligible
	rule, ok := <- j.NextJob
	
	//Si le chan NextJob est closed = plus de règles dispo
	if !ok {
		log.Println("No more job for the slave")
		return errors.New("No more job")
	}

	//On récupère le Node
	n := j.Rules[rule]
	
	//MAJ du statut
	j.UpdateNode.Lock()
	n.Affected = true
	j.UpdateNode.Unlock()

	log.Println("Slave get a new target:", n.Target)
	
	//On envoie le Node au slave
	*res = *n 
	
	return nil
}


//GetFile for RPC
func (j *Job) GetFile(filename string, res *File) error {
	
	f, err := GetFile(filename)
	*res = f

	return err
}


//Normal GetFile
func GetFile(filename string) (File, error) {
	var f File

	//Si le fichier existe
	s, err := os.Stat(filename);
	if os.IsNotExist(err) {
		return f, err
	}

	//Copie du fichier
	f.Content, err = ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Impossible to read file:", filename)
	}
	f.Name = filename
	f.Mode = s.Mode()

	//On envoi le fichier
	return f, nil
}


func (j *Job) FinishJob(s SlaveResponse, i *int) error {
	log.Println("Slave wants to finish a job")

	
	if len(s.File.Content) > 0 {
		//On écrit le fichier de résultat du job
		if err := ioutil.WriteFile(s.File.Name, s.File.Content, s.File.Mode); err != nil {
			return err
		}
		log.Println("Write reply file from the slave:", s.File.Name)
	}


	for _, out := range s.Output {
		fmt.Println(out)
	}

	//On récupère le Node
	n := j.Rules[s.Target]
	
	//MAJ du statut
	j.UpdateNode.Lock()
	n.Affected = false
	n.Done = true
	j.UpdateNode.Unlock()

	log.Println("Target:", n.Target, " DONE")

	//Si c'est la dernière tâche on envoie le signal de fin
	if n.Target == j.MainTarget {
		log.Println("Last target DONE, closing server")
		j.Done <- 1
	} else if j.Wait {
		log.Println("Allow wake up")
		j.WakeUp <- 1
	}

	return nil
}