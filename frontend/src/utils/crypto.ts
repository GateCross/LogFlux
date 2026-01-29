import CryptoJS from 'crypto-js';

const KEY = 'logflux_secret_k'; // 16 bytes key

/**
 * Encrypt string
 * @param str String to encrypt
 */
export function encrypt(str: string) {
    const key = CryptoJS.enc.Utf8.parse(KEY);
    const iv = CryptoJS.enc.Utf8.parse(KEY); // Use key as IV for simplicity
    const srcs = CryptoJS.enc.Utf8.parse(str);
    const encrypted = CryptoJS.AES.encrypt(srcs, key, {
        iv: iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7
    });
    return encrypted.toString();
}

/**
 * Decrypt string
 * @param str String to decrypt
 */
export function decrypt(str: string) {
    const key = CryptoJS.enc.Utf8.parse(KEY);
    const iv = CryptoJS.enc.Utf8.parse(KEY);
    const decrypt = CryptoJS.AES.decrypt(str, key, {
        iv: iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7
    });
    return CryptoJS.enc.Utf8.stringify(decrypt).toString();
}
