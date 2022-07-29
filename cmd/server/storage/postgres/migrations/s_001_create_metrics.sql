CREATE TABLE metrics (  
  id    VARCHAR ( 50 ) NOT NULL,
  hash  VARCHAR ( 65 ),
  target  VARCHAR ( 50 ) NOT NULL,
  mtype VARCHAR ( 50 ) NOT NULL,
  mdelta BIGINT CHECK (
    	(mtype = 'counter' AND mdelta IS NOT NULL)
      OR (mtype = 'gauge' AND mvalue IS NOT NULL)
  ),
  mvalue REAL CHECK (
    	(mtype = 'counter' AND mdelta IS NOT NULL)
      OR (mtype = 'gauge' AND mvalue IS NOT NULL)
  ),
  PRIMARY KEY (id, target)
);
