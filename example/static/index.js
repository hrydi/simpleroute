(function (){
    'use strict'

    /**
     * 
     * @param {string} url 
     * @param {?RequestInit} opt 
     * @returns 
     */
    async function Request(url, opt = null) {
        try {
            return await fetch(url, opt)
        } catch (e) {
            return Promise.reject(e)
        }
    }

    const pEl = document.getElementsByTagName('p')
    for(const el of pEl) {
        el.style = `background-color: red; color: white; padding: 2pt; font-weight: bold;`
    }

    console.info(`loaded index.js`)

    Request("/user/profile", {
        method: "GET",
        headers: {
            "X-Api-Key": Math.floor(Math.random() * 100)
        }
    })
    .then(res => res.json())
    .then(console.info)
    .catch(console.error)
})()