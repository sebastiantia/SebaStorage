package main


import(
	"fmt"
	"os"
	"encoding/json"
	"sync"
	"io/ioutil"
	"path/filepath"
	"github.com/jcelliott/lumber"
)

const Version = "1.0.0"

type (
	Logger interface{
		Fatal(string, ...interface{})
		Error(string, ... interface{})
		Warn(string, ...interface{})
		Info(string, ...interface{})
		Debug(string, ...interface{})
		Trace(string, ...interface{})
	}

	Driver struct {//responsible for interacting code w db
		mutex 	sync.Mutex //mutexes to write and delete
		mutexes map[string]*sync.Mutex
		dir 	string
		log 	Logger
	}
)

type Options struct{
	Logger
}

type Address struct {
	City string
	State string
	Country string
	Pincode json.Number
}
//allows user to group/combine items of possibly different types into a single type
//JSON used for db, but struct is what golang understands

type User struct {
	Name string
	Age json.Number
	Contact string
	Company string
	Address Address
}

func New(dir string, options *Options) (*Driver, error) {
	dir = filepath.Clean(dir)

	opts := Options{}

	if options != nil {
		opts = *options
	}

	if opts.Logger == nil {
		opts.Logger = lumber.NewConsoleLogger((lumber.INFO))
	}

	driver := Driver{
		dir:     dir,
		mutexes: make(map[string]*sync.Mutex),
		log:     opts.Logger,
	}

	if _, err := os.Stat(dir); err == nil {
		opts.Logger.Debug("Using '%s' (database already exists)\n", dir)
		return &driver, nil
	}

	opts.Logger.Debug("Creating the database at '%s'...\n", dir)
	return &driver, os.MkdirAll(dir, 0755)
}

func checkDir(path string)(fi os.FileInfo, err error){ //checks if dictionary exists
	if fi, err = os.Stat(path); os.IsNotExist(err){
		fi, err = os.Stat(path + ".json")
	}

	return 
}

//struct methods
func (d *Driver) Write(collection, resource string, v interface{}) error{
	//we create the .json file
	if collection == "" {
		return fmt.Errorf("Missing colllection - no place to save record!")
	}
	if resource == "" {
		return fmt.Errorf("Missing resource - no resource to save on (no name)!")
	}
	mutex := d.getOrCreateMutex(collection)

	mutex.Lock()
	defer mutex.Unlock() //only when the function is completed

	//going to the directory, and joining the collection to it to assign proper FILE PATH
	dir := filepath.Join(d.dir, collection)
	fnlPath := filepath.Join(dir, resource+ ".json")
	tmpPath := fnlPath + ".tmp"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	//converting everything from v interface to json and we include assign to b
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}
	//everything is written with \n
	b = append(b, byte('\n'))
	//we write in the .json file

	if err := ioutil.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}

	//moves oldpath to newpath. If newpath already exists and is not a directory, rename replaces it
	return os.Rename(tmpPath, fnlPath)
}



func (d *Driver) Read(collection, resource string, v interface{}) error {

	if collection == ""{
		return fmt.Errorf("Missing collection - unable to read!")
	}

	if resource == ""{
		return fmt.Errorf("Missing resource - no resource to read record (no name)!")
	}
	
	record := filepath.Join(d.dir, collection, resource)

	if _, err := checkDir(record); err != nil {
		return err
	}
	//reading from db
	b, err := ioutil.ReadFile(record + ".json")

	if err != nil {
		return err
	}
	//Unmarshalled response from the read function
	return json.Unmarshal(b, &v)
}

//returns as a slice of strings
func (d *Driver) ReadAll(collection string) ([]string, error) {

	if collection == "" {
		return nil, fmt.Errorf("Missing collection - unable to read!")
	}

	dir := filepath.Join(d.dir, collection)

	if _, err := checkDir(dir); err != nil {
		return nil, err
	}
	//ioutil is a very common package with golang
	//reading the entire directory with multiple json files
	files, _ := ioutil.ReadDir(dir)

	var records []string

	//iterator and value
	//to access particular files, we loop through all dir's file name

	for _, file := range files{
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		records = append(records, string(b))
	}

	return records, nil


}

func (d* Driver) Delete(collection , resource string) error {
	path := filepath.Join(collection, resource)

	mutex :=d.getOrCreateMutex(collection)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(d.dir, path)

	switch fi, err := checkDir(dir);{
	case fi==nil, err!= nil:
		return fmt.Errorf("Unable to find file or directory name %v\n", path) 
	
	//removing entire folder
	case fi.Mode().IsDir():
		return os.RemoveAll(dir)

	//removing all files inside folder
	case fi.Mode().IsRegular():
		return os.RemoveAll(dir + ".json")
	}
	return nil
	
}


//mutual exlucsion lock = mutex
func (d* Driver) getOrCreateMutex(collection string) *sync.Mutex{
	//mutexes are a map
	d.mutex.Lock()
	defer d.mutex.Unlock()
	m, exist := d.mutexes[collection]
	if !exist {
		m = &sync.Mutex{} //basically create an empty mutex
		d.mutexes[collection] = m
	}

	return m
}


func main(){
	dir := "./"
	
	db, err := New(dir, nil)
	
	if err != nil {
		fmt.Println("Error: ", err)
	}

	employees := []User{
			{"Seb","23","17781234567","VTS", Address{"Waterloo", "Ontario","Canada","234234"}},
			{"Charles","23","17781234567","PW", Address{"Vancouver", "British Columbia","Canada","234234"}},
			{"Supreme Leader","23","17781234567","Stay home", Address{"Vancouver", "British Columbia","Canada","234234"}},
			{"Buc","23","1778621234567","Watching anime", Address{"Vancouver", "British Columbia","Canada","234234"}},
	}


	for _, value := range employees {
		db.Write("users", value.Name, User{
			Name:    value.Name,
			Age:     value.Age,
			Contact: value.Contact,
			Company: value.Company,
			Address: value.Address,
		})
	}

	records, err := db.ReadAll("users")
	if err != nil {
		fmt.Println("Error", err)
	}
	fmt.Println(records)

	allusers := []User{}

	for _, f := range records {
		employeeFound := User{}
		if err := json.Unmarshal([]byte(f), &employeeFound); err != nil {
			fmt.Println("Error", err)
		}
		allusers = append(allusers, employeeFound)
	}
	fmt.Println((allusers))
	// if err := db.Delete("user", "john"); err != nil {
	// 	fmt.Println("Error: ", err)
	// }

	// if err := db.Delete("user", ""); err != nil {
	// 	fmt.Println("Error: ", err)
	// }




}

