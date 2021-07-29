print('Start #################################################################');

test = db.getSiblingDB('test_db');
test.createUser(
  {
    user: 'test',
    pwd: 'test',
    roles: [{ role: 'readWrite', db: 'test_db' }],
  },
);

test.createCollection('reps');
db.reps.createIndex({address: "text"}, {unique: true})


print('END #################################################################');