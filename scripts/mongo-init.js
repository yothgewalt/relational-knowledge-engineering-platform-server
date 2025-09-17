db = db.getSiblingDB('relational_knowledge_db');

db.createUser({
  user: 'app_user',
  pwd: 'app_password123',
  roles: [
    {
      role: 'readWrite',
      db: 'relational_knowledge_db'
    }
  ]
});

db.createCollection('documents');
db.createCollection('graphs');
db.createCollection('processing_logs');

db.documents.createIndex({ "filename": 1 });
db.documents.createIndex({ "uploaded_at": 1 });
db.graphs.createIndex({ "document_id": 1 });
db.graphs.createIndex({ "created_at": 1 });

print('Database initialization completed successfully');