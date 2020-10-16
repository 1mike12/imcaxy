# Imcaxy - the imaginary cache and proxy service

This project is under development.

Expected functionality:

- [ ] simple proxy for imaginary service
- [ ] cache imaginary results in minio block storage, save cache info inside mongo database
- [ ] scan for changes in files, if some file is changed, discard all cached responses of this file
- [ ] maximum free space of disk monitoring
- [ ] support three units of disk space usage:
  - [ ] GB - amount of gigabytes that can be used
  - [ ] MB - amount of megabytes that can be used
  - [ ] % - percent of free space that can be used
- [ ] if system uses more than maximum value of disk space, it scans for the most less used cached images and removes them, then starts to cache the newest used element
- [ ] collects statistics about resources usage, including:
  - [ ] how frequently is selected resource used
  - [ ] last time when was resource was used
- [ ] from time to time scans for unknown cached data and unknown database entries
- [ ] two working modes:
  - [ ] imcaxy and imaginary have access to project files, does not need to send images to imaginary worker because imaginary already has own copy of files
  - [ ] imcaxy has access to project files and imaginary worker does not, imcaxy sends images to imaginary over http
- [ ] currently processed files registry with process parameters, if file we want to optimize is already processing (and process parameters are the same) it should wait for end of optimization process and get the request from minio server instead of sending two or more same requests to imaginary worker
