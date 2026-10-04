const CROCKFORD = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

/**
 * A ULID: 48 bits of milliseconds, then 80 random bits, in Crockford base32.
 * IDs made offline on two devices never collide.
 */
export function ulid(now: number = Date.now()): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes.subarray(6));
  let t = BigInt(now);
  for (let i = 5; i >= 0; i--) {
    bytes[i] = Number(t & 0xffn);
    t >>= 8n;
  }
  let n = 0n;
  for (const b of bytes) {
    n = (n << 8n) | BigInt(b);
  }
  let out = "";
  for (let i = 0; i < 26; i++) {
    out = CROCKFORD.charAt(Number(n & 31n)) + out;
    n >>= 5n;
  }
  return out;
}
