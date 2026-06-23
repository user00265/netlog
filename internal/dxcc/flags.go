package dxcc

// adifToISO2 maps an ADIF DXCC entity number to an ISO 3166-1 alpha-2 code (or a
// flag-icons subdivision code such as gb-eng) used to render a country flag.
//
// DXCC entities do not map one-to-one to ISO countries: many are islands,
// territories, or research areas. Each entry resolves to the most appropriate
// flag-icons code; dependencies use their own code when one exists, otherwise
// the administering country's. A handful of non-geographic entities (HQ orgs,
// disputed reefs) intentionally have no flag and fall back to a placeholder in
// the UI. The numbers were taken from the loaded cty.xml so they match the
// dataset exactly.
var adifToISO2 = map[int]string{
	1: "ca", 3: "af", 4: "mu", 5: "ax", 6: "us", 7: "al", 9: "as",
	10: "tf", 11: "in", 12: "ai", 13: "aq", 14: "am", 15: "ru", 16: "nz",
	17: "ve", 18: "az", 20: "us", 21: "es", 22: "pw", 24: "bv", 27: "by",
	29: "es", 31: "ki", 32: "es", 33: "io", 34: "nz", 35: "cx", 36: "fr",
	37: "cr", 38: "cc", 40: "gr", 41: "tf", 43: "us", 45: "gr", 46: "my",
	47: "cl", 48: "ki", 49: "gq", 50: "mx", 51: "er", 52: "ee", 53: "et",
	54: "ru", 56: "br", 60: "bs", 61: "ru", 62: "bb", 63: "gf", 64: "bm",
	65: "vg", 66: "bz", 69: "ky", 70: "cu", 71: "ec", 72: "do", 74: "sv",
	75: "ge", 76: "gt", 77: "gd", 78: "ht", 79: "gp", 80: "hn", 82: "jm",
	84: "mq", 86: "ni", 88: "pa", 89: "tc", 90: "tt", 91: "aw", 94: "ag",
	95: "dm", 96: "ms", 97: "lc", 98: "vc", 99: "tf", 100: "ar", 103: "gu",
	104: "bo", 105: "us", 106: "gg", 107: "gn", 108: "br", 109: "gw", 110: "us",
	111: "hm", 112: "cl", 114: "im", 116: "co", 118: "sj", 120: "ec", 122: "je",
	123: "us", 124: "tf", 125: "cl", 126: "ru", 129: "gy", 130: "kz", 131: "tf",
	132: "py", 133: "nz", 135: "kg", 136: "pe", 137: "kr", 138: "us", 140: "sr",
	141: "fk", 142: "in", 143: "la", 144: "uy", 145: "lv", 146: "lt", 147: "au",
	148: "ve", 149: "pt", 150: "au", 152: "mo", 153: "au", 157: "nr", 158: "vu",
	159: "mv", 160: "to", 161: "co", 162: "nc", 163: "pg", 165: "mu", 166: "mp",
	167: "ax", 168: "mh", 169: "yt", 170: "nz", 171: "au", 172: "pn", 173: "fm",
	174: "us", 175: "pf", 176: "fj", 177: "jp", 179: "md", 180: "gr", 181: "mz",
	182: "us", 185: "sb", 187: "ne", 188: "nu", 189: "nf", 190: "ws", 191: "ck",
	192: "jp", 195: "gq", 197: "us", 199: "aq", 201: "za", 202: "pr", 203: "ad",
	204: "mx", 205: "sh", 206: "at", 207: "mu", 209: "be", 211: "ca", 212: "bg",
	213: "mf", 214: "fr", 215: "cy", 216: "co", 217: "cl", 219: "st", 221: "dk",
	222: "fo", 223: "gb-eng", 224: "fi", 225: "it", 227: "fr", 230: "de", 232: "so",
	233: "gi", 234: "ck", 235: "gs", 236: "gr", 237: "gl", 238: "aq", 239: "hu",
	240: "gs", 241: "aq", 242: "is", 245: "ie", 248: "it", 249: "kn", 250: "sh",
	251: "li", 252: "ca", 253: "br", 254: "lu", 256: "pt", 257: "mt", 259: "sj",
	260: "mc", 262: "tj", 263: "nl", 265: "gb-nir", 266: "no", 269: "pl", 270: "tk",
	272: "pt", 273: "br", 274: "sh", 275: "ro", 276: "tf", 277: "pm", 278: "sm",
	279: "gb-sct", 280: "tm", 281: "es", 282: "tv", 283: "gb", 284: "se", 285: "vi",
	286: "ug", 287: "ch", 288: "ua", 289: "un", 291: "us", 292: "uz", 293: "vn",
	294: "gb-wls", 295: "va", 296: "rs", 297: "us", 298: "wf", 299: "my", 301: "ki",
	302: "eh", 303: "au", 304: "bh", 305: "bd", 306: "bt", 308: "cr", 309: "mm",
	312: "kh", 315: "lk", 318: "cn", 321: "hk", 324: "in", 327: "id", 330: "ir",
	333: "iq", 336: "il", 339: "jp", 342: "jo", 344: "kp", 345: "bn", 348: "kw",
	354: "lb", 363: "mn", 369: "np", 370: "om", 372: "pk", 375: "ph", 376: "qa",
	378: "sa", 379: "sc", 381: "sg", 382: "dj", 384: "sy", 386: "tw", 387: "th",
	390: "tr", 391: "ae", 400: "dz", 401: "ao", 402: "bw", 404: "bi", 406: "cm",
	408: "cf", 409: "cv", 410: "td", 411: "km", 412: "cg", 414: "cd", 416: "bj",
	420: "ga", 422: "gm", 424: "gh", 428: "ci", 430: "ke", 432: "ls", 434: "lr",
	436: "ly", 438: "mg", 440: "mw", 442: "ml", 444: "mr", 446: "ma", 450: "ng",
	452: "zw", 453: "re", 454: "rw", 456: "sn", 458: "sl", 460: "fj", 462: "za",
	464: "na", 466: "sd", 468: "sz", 470: "tz", 474: "tn", 478: "eg", 480: "bf",
	482: "zm", 483: "tg", 489: "fj", 490: "ki", 492: "ye", 497: "hr", 499: "si",
	501: "ba", 502: "mk", 503: "cz", 504: "sk", 505: "tw", 507: "sb", 508: "pf",
	509: "pf", 510: "ps", 511: "tl", 512: "nc", 513: "pn", 514: "me", 515: "as",
	516: "bl", 517: "cw", 518: "sx", 519: "bq", 520: "bq", 521: "ss", 522: "xk",
}

// flagFor returns the flag-icons code for a DXCC ADIF entity, or "" when no
// appropriate flag exists (non-geographic or disputed entities).
func flagFor(adif int) string {
	return adifToISO2[adif]
}
