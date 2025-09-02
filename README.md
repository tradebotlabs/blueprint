# Forex Platform Blueprint Service

## Owner: JeelRupapara (zeelrupapara@gmail.com) 

This is Forex Platform GoLang micro-service project 

1. Must follow the code structure as per lead developer, this code can't be copied or saved for personal use 
2. Unless a good reason or bug in this blueprint no allowed allowed to change 



# Forex Platform Blueprint 

A blueprint micro-service 

Rename this to the new services as given by JIRA project for example 

## git name 
blueprint-svc.git 

## JIRA issues branch 

when a JIRA ticket features 

branch name with the ticket is created 

git -b blueprint-svc-BUG-XXX.git or feature git -b blueprint-svc-STORY-XXX-.git

if the test approved then merge to master 

## merge 

git checkout master 
git merge blueprint-svc-BUG-XXX.git


Remove all comments before doing anything 
Run the services before any code

$ go run cmd/main.go

if not working STOP & FIX or Ask for help 


# blueprint 

A service that provide the following functions as gPRC handlers :

1. Call() : Will return  a name message 
2. 

## Testing 
each 


## Docker 

Every micro service must run in a docker container to that
the blueprint have a DockerFile which will give you how
you will make a images, we need to make sure that
the service will run without issues so we use Docker stage build 

When we finish from Sprint we need to push images to our registry 

## Docker compose 

In order to test all other micro service we will run 
one full using docker-compose.yaml, example  given in
blueprint folder

$ sudo docker compose up -d 

the command will run your development stack 


### Development Stack 

During testing some service need to run like Redis, MySQL ..etc 
include all your needs in stack.yaml and run for development the docker compose command

$ sudo docker compose -f stack.yaml up -d


### ENV

All setting will go into .ENV file the setting need to be
static and dynamic, static values that is important to run the stack
dynamic values that is used for docker compose 

* Static values 

GPRC_HOST=127.0.01
GRPC_PORT=3000

* Dynamic values, this is the name in docker  network not in our running host 

MYSQL_HOST=mysql



