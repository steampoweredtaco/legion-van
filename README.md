# We are legion, how's my hair?
![We are Legion](https://raw.githubusercontent.com/steampoweredtaco/legion-van/main/assets/legion-van-demo.gif)
legion-van is a potassium rich banano visual monKey vanity generator for your terminal. Find your perfect monkey and look good while doing it.

Don't know what banano is? [Go find out!](https://banano.cc/)  
Don't know what a monKey is. [Go look!](https://monkey.banano.cc/)  
Want to talk with us more about banano? [Join Us!](https://chat.banano.cc/)

# Features
* Find any **official** visual monKey representation given enough time and produce the private banano wallet seed, picture, and json stats about found monKey.
* A gui in a terminal, say what now?
* Can output svg or png files of found monKey.
* Fine tune of cpu usage and network utilization for testing monKeys.
* Lots of potassium.
* Attempts of humor.

# Upcomming Features
* Display the odds of finding monKey with supplied filter.
* Pipe nano vanity generator output into command and test for monKey. This provides a way to get a text and visual vanity, but probably a bit slowly.

# TODO to get out of Beta (pull requests welcome)
* Compiling and working tests for most utility functions and the engine.
* All executable and docker environments for platforms.
* Option to find ad-hoc account seeds.
* Likely better documentation.

# Install
The recommended way to run this is via pulling this repo and running directly with go.
You should inspecting the source anyway and trusting what you build because this application
requires the internet and it is going to be generating private keys that need to be kept safe.

## Requirements
* Any standard shell or terminal.
* git [Github instrunctions](https://docs.github.com/en/get-started/quickstart/set-up-git)
* go 1.16+ [Instrunctions from golang.org](https://golang.org/doc/install)

## Steps
Pull this repository:

`git clone https://github.com/steampoweredtaco/legion-van.git`  
`cd legion-van`  
```bash
taco:~$ git clone https://github.com/steampoweredtaco/legion-van.git
Cloning into 'legion-van'...
remote: Enumerating objects: 298, done.
remote: Counting objects: 100% (298/298), done.
remote: Compressing objects: 100% (154/154), done.
remote: Total 298 (delta 132), reused 270 (delta 104), pack-reused 0
Receiving objects: 100% (298/298), 98.21 KiB | 1.89 MiB/s, done.
Resolving deltas: 100% (132/132), done.
```

Inspect the source to confirm it only contacts https://monkey.banano.cc so you can trust this app to generate keys.

Build, this will put an executable called `legion-van` in the current directory:
`go build cmd/legion-van/legion-van.go`
```bash
taco:~/legion-van$ go build cmd/legion-van/legion-van.go                                                                           taco:~/legion-van$ ls                                                                                                              LICENSE  README.md  assets  bananoutils  cmd  engine  go.mod  go.sum  gui  image  legion-van  scripts      
```

Run the executable, use --help for more info and some examples.  
`legion-van --help`
```bash
taco:~/legion-van$ ./legion-van --help
Usage:
  legion-van [OPTIONS]

Application Options:
      --duration=              How long to run the search for (default: 1m)
      --max_requests=          Maxiumum outstanding parallel requests to monkeyapi (default: 4)
      --disable_review         Disable the gui and preview of monkeys
      --image_format=[png|svg] Set the target image format for saving monkey found in options are svg or png. svg is faster (default:
                               png)
      --batch_size=            Number of monkeys to test per batch request, higher or lower may affect performance (default: 2500)
      --debug                  Changes logging and makes terminal virtual for debugging issues.
      --verbose                Changes logging to print debug.
      --monkey_api=            To change the backend monkey server, defaults to the official one. (default: https://monkey.banano.cc)
      --threads=               Changes number of threads to use, defaults to 2, with a decent machine this is probably all you need.
                               Set to -1 for all hardware cpu threads available. (default: 2)
  -g, --nogui                  Do not use a terminal gui just give you the straight banano.

Vanity Filters:
  -V, --help-vanity
  -H=                          hat option. See --help-vanity for list
  -G=                          glasses option. See --help-vanity for list
  -O=                          mouth option. See --help-vanity for list
  -C=                          cloths option. See --help-vanity for list
  -F=                          feet option. See --help-vanity for list
  -T=                          tail option. See --help-vanity for list
  -M=                          misc  option. See --help-vanity for list

Help Options:
  -h, --help                   Show this help message
  ```
# Examples
This will search for monkie's with beanies that have the banano on it for 10 seconds:  
`./legion-van -H beanie-ban --duration=10s`

The following options would require a flamethrower and cap of any sort:  
`./legion-van -M flamethrower -H cap`

The following options would require a flamethrower **OR** camera and a cap of any sort:  
`./legion-van -M flamethrower -M camera -H cap`

The following requires a monKey with either a flamethrower or camera
**AND** a cap or beanie:  
`./legion-van -M flamethrower -M camera -C cap -C beanie`

Each option choice maybe abbrevated to match the more general option.  For example, to get money's that match a pink tie:  
`./legion-van -M flamethrower -M tie-pink`  
But to get any color tie:  
`./legion-van -M flamethrower -M tie`

See `./legion-van --help-vanity for more examples`
# Troubleshooting
**MonKeys look ghostly**
```
Make sure you have a sane terminal, try `export TERM=xterm-256color` or the equivelant for your platform before running legion-van
``` 

# FAQ
### ***Where are my monKeys keys store**
By default it is in the directory `./fundMonKeys` where the `./legion-van` command was ran. For convince in the case of multiple finds, a named image of the monkey in .png or .svg format is saved so you can quickly distinguish which same named .json version of the file has your private key.

### **How can I show my appreciation?**

`Make me your primary representive or donate me some ban; use this address for both:`

![ban_3tacocatezozswnu8xkh66qa1dbcdujktzmfpdj7ax66wtfrio6h5sxikkep](https://raw.githubusercontent.com/steampoweredtaco/legion-van/main/assets/tacocatrepQR.png)