db.auth('root', 'root');
db = db.getSiblingDB('accountant');
db.createUser(
    {
        user: "accountant",
        pwd: "accountant",
        roles: [
            {
                role: "readWrite",
                db: "accountant"
            }
        ]
    }
);
db = db.getSiblingDB('transactor');
db.createUser(
    {
        user: "transactor",
        pwd: "transactor",
        roles: [
            {
                role: "readWrite",
                db: "transactor"
            }
        ]
    }    
);
