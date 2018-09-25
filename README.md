# openmcdf
**Structured Storage**

OpenMCDF developers to manipulate [Microsoft Compound Document Files](https://msdn.microsoft.com/en-us/library/dd942138.aspx) (also known as OLE structured storage).

Compound file includes multiple streams of information (document summary, user data) in a single container.

This file format is used under the hood by a lot of applications: all the documents created by Microsoft Office until the 2007 product release are structured storage files. Windows thumbnails cache files (thumbs.db) are compound documents as well as .msg Outlook messages. Visual Studio .suo files (solution options) are compound files and a lot of audio/video editing tools save project file in a compound container.

OpenMcdf supports read/write operations on streams and storages and traversal of structures tree. It supports version 3 and 4 of the specifications, uses lazy loading wherever possible to reduce memory usage and offer an intuitive API to work with structured files.
