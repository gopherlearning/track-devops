CREATE TABLE metrics (  
  target VARCHAR ( 50 ) UNIQUE NOT NULL,
  data jsonb NOT NULL
);
