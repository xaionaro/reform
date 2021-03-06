CREATE TABLE people (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  group_id int DEFAULT 65534,
  name varchar NOT NULL,
  email varchar,
  created_at datetime NOT NULL,
  updated_at datetime
);

CREATE TABLE projects (
  name varchar NOT NULL,
  id varchar PRIMARY KEY NOT NULL,
  start date NOT NULL,
  end date
);

CREATE TABLE person_project (
  person_id integer NOT NULL REFERENCES people ON DELETE CASCADE,
  project_id varchar NOT NULL REFERENCES projects ON DELETE CASCADE,
  UNIQUE (person_id, project_id)
);

CREATE TABLE id_only (
  id integer NOT NULL PRIMARY KEY AUTOINCREMENT
);
