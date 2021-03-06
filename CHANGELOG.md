# Tide-whisperer

Data access API for tidepool

## 0.9.1 - 2021-07-01
### Fixed
- YLP-858 Some algorithm settings are visible on the daily view
- YLP-859 Missing index on id field for deviceData

## 0.9.0 - 2021-06-11
### Added
- YLP-747 New data/v1 routes
### Fixed
- YLP-819 performance issues on production environment


## 0.8.1 - 2020-03-11
### Engineering Use
- Change build server from Travis to Jenkins

## 0.8.0 - 2020-03-08
### Changed
- YLP-471 Implement authorization rules for tide-whisperer

## 0.7.4 - 2020-11-04
### Engineering
- Review buildDoc to match "dblp" release tags and ensure copy latest is done

## 0.7.3 - 2020-11-03
### Fixed
- YLP-255 MongoDb connection issue

## 0.7.2 - 2020-10-29
### Engineering
- YLP-245 Review openapi generation so we can serve it through a website

## 0.7.1 - 2020-09-25
### Fixed
- Fix S3 deployment

## 0.7.0 - 2020-09-16
### Changed
- PT-1441 Tide-Whisperer should be able to start without MongoDb

## 0.6.1 - 2020-08-04
### Engineering
- PT-1446 Generate SOUP document

## 0.6.0 - 2020-04-23
### Added 
- PT-1193 New API access point : compute time in range data for a set of users (last 24 hours)

## 0.5.2 - 2020-04-14
### Engineering
- PT-1232 Integrate latest changes from Tidepool develop branch
- PT-1034 Review API structure
- PT-1005 Openapi documentation

## 0.5.1 - 2020-03-26
### Fixed
- PT-1220 ReservoirChange objects are not retrieved

## 0.5.0 - 2020-03-19
### Changed
- PT-1150 Add filter on parameter level based on model

## 0.4.0 - 2019-10-28 
### Added 
- PT-734 Display the application version number on the status endpoint (/status).

## 0.3.2 
### Fixed 
- PT-649 Get Level 2 and 3 parameters for parameter history

## 0.3.1
### Added
- PT-607 DBLHU users should access to Level 1 and Level 2 parameters in the parameters history.

## 0.3.0
### Added
- PT-511 Access diabeloop system parameters history from tide-whisperer

## 0.2.0 
### Added
- Integration from Tidepool latest changes

  Need to provide a new configuration item _auth_ in _TIDEPOOL_TIDE_WHISPERER_ENV_  (see [.vscode/launch.json.template](.vscode/launch.json.template) or [env.sh](env.sh) for example)

### Changed
- Update to MongoDb 3.6 drivers in order to use replica set connections. 

## 0.1.2 - 2019-04-17

### Changed
- Fix status response of the service. On some cases (MongoDb restart mainly) the status was in error whereas all other entrypoints responded.

## 0.1.1 - 2019-01-28

### Changed
- Remove dependency on lib-sasl

## 0.1.0 - 2019-01-28

### Added
- Add support to MongoDb Authentication

## 0.0.1 - 2018-06-28

### Added
- Enable travis CI build 
