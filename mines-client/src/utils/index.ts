function  isDevMode() {
    // @ts-ignore
    return process.env.NODE_ENV === 'development';
}

let host = window.location.hostname
let port = window.location.port
if (isDevMode()){
    host = 'localhost'
    port = '8080'
}

export { host, port };

export { isDevMode };