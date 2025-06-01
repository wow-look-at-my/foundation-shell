# Core library - shared between main program and platform implementations
add_library(core STATIC
    GlobalDispatcher.cpp
)

target_link_libraries(core PUBLIC
    mh-stuff
)