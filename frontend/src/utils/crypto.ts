import CryptoJS from 'crypto-js';

const KEY = 'LogFlux@AES#2024'; // 16 bytes key - 与后端 config.yaml 中的 AESKey 一致

/**
 * Encrypt string using AES-CBC with random IV
 * IV is prepended to the ciphertext
 * @param str String to encrypt
 */
export function encrypt(str: string) {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  // 生成随机 IV
  const iv = CryptoJS.lib.WordArray.random(16);
  const srcs = CryptoJS.enc.Utf8.parse(str);
  const encrypted = CryptoJS.AES.encrypt(srcs, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  });
  // 将 IV 和密文拼接，然后 base64 编码
  const result = iv.concat(encrypted.ciphertext);
  return CryptoJS.enc.Base64.stringify(result);
}

/**
 * Decrypt string using AES-CBC
 * IV is prepended to the ciphertext
 * @param str String to decrypt
 */
export function decrypt(str: string) {
  const key = CryptoJS.enc.Utf8.parse(KEY);
  // 解析 base64，分离 IV 和密文
  const ciphertext = CryptoJS.enc.Base64.parse(str);
  const iv = CryptoJS.lib.WordArray.create(ciphertext.words.slice(0, 4));
  const encrypted = CryptoJS.lib.WordArray.create(ciphertext.words.slice(4));
  const cipherParams = CryptoJS.lib.CipherParams.create({ ciphertext: encrypted });
  const decrypted = CryptoJS.AES.decrypt(cipherParams, key, {
    iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  });
  return CryptoJS.enc.Utf8.stringify(decrypted).toString();
}
