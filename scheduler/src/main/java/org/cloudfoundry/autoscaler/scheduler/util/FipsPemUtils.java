package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.StringReader;
import java.security.KeyFactory;
import java.security.NoSuchAlgorithmException;
import java.security.NoSuchProviderException;
import java.security.PrivateKey;
import java.security.cert.CertificateException;
import java.security.cert.CertificateFactory;
import java.security.cert.X509Certificate;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.PKCS8EncodedKeySpec;
import org.bouncycastle.asn1.ASN1Sequence;
import org.bouncycastle.asn1.pkcs.PKCSObjectIdentifiers;
import org.bouncycastle.asn1.pkcs.PrivateKeyInfo;
import org.bouncycastle.asn1.pkcs.RSAPrivateKey;
import org.bouncycastle.asn1.x509.AlgorithmIdentifier;
import org.bouncycastle.asn1.x9.X9ObjectIdentifiers;
import org.bouncycastle.util.io.pem.PemObject;
import org.bouncycastle.util.io.pem.PemReader;

public class FipsPemUtils {

  private static final String BCFIPS_PROVIDER = "BCFIPS";
  private static volatile java.security.Provider cachedProvider;

  private FipsPemUtils() {}

  public static X509Certificate parseCertificate(String pem) throws CertificateException {
    java.util.List<X509Certificate> chain = parseCertificateChain(pem);
    return chain.get(0);
  }

  /**
   * Parses all X.509 certificates from a PEM string that may contain multiple concatenated
   * CERTIFICATE blocks (e.g. a leaf cert followed by intermediates, as in CF_INSTANCE_CERT).
   *
   * @return list of certificates in order: [leaf, intermediate, ...]
   */
  @SuppressWarnings("unchecked")
  public static java.util.List<X509Certificate> parseCertificateChain(String pem)
      throws CertificateException {
    if (pem == null || pem.isBlank()) {
      throw new CertificateException("PEM input is null or empty");
    }
    CertificateFactory factory = CertificateFactory.getInstance("X.509", getCryptoProvider());
    java.util.Collection<X509Certificate> certs =
        (java.util.Collection<X509Certificate>)
            (java.util.Collection<?>)
                factory.generateCertificates(new ByteArrayInputStream(pem.getBytes(java.nio.charset.StandardCharsets.UTF_8)));
    if (certs.isEmpty()) {
      throw new CertificateException("No certificates found in PEM input");
    }
    return new java.util.ArrayList<>(certs);
  }

  public static PrivateKey parsePrivateKey(String pem)
      throws NoSuchAlgorithmException, InvalidKeySpecException, NoSuchProviderException,
          IOException, IllegalArgumentException {
    if (pem == null || pem.isBlank()) {
      throw new IllegalArgumentException("PEM input is null or empty");
    }
    PemObject pemObject;
    try (PemReader pemReader = new PemReader(new StringReader(pem))) {
      pemObject = pemReader.readPemObject();
    }
    if (pemObject == null) {
      throw new IllegalArgumentException("No PEM object found in input");
    }

    byte[] derBytes = pemObject.getContent();
    String pemType = pemObject.getType();

    if ("RSA PRIVATE KEY".equals(pemType)) {
      RSAPrivateKey rsaKey = RSAPrivateKey.getInstance(derBytes);
      AlgorithmIdentifier rsaAlgId =
          new AlgorithmIdentifier(PKCSObjectIdentifiers.rsaEncryption);
      PrivateKeyInfo keyInfo = new PrivateKeyInfo(rsaAlgId, rsaKey);
      derBytes = keyInfo.getEncoded();
    } else if ("EC PRIVATE KEY".equals(pemType)) {
      org.bouncycastle.asn1.sec.ECPrivateKey ecKey =
          org.bouncycastle.asn1.sec.ECPrivateKey.getInstance(derBytes);
      AlgorithmIdentifier ecAlgId =
          new AlgorithmIdentifier(X9ObjectIdentifiers.id_ecPublicKey, ecKey.getParameters());
      PrivateKeyInfo keyInfo = new PrivateKeyInfo(ecAlgId, ecKey);
      derBytes = keyInfo.getEncoded();
    }

    PKCS8EncodedKeySpec keySpec = new PKCS8EncodedKeySpec(derBytes);
    String algorithm = detectKeyAlgorithm(derBytes);
    KeyFactory keyFactory;
    if (isFipsProviderAvailable()) {
      keyFactory = KeyFactory.getInstance(algorithm, BCFIPS_PROVIDER);
    } else {
      keyFactory = KeyFactory.getInstance(algorithm);
    }
    return keyFactory.generatePrivate(keySpec);
  }

  private static String detectKeyAlgorithm(byte[] pkcs8Bytes) {
    try {
      PrivateKeyInfo keyInfo = PrivateKeyInfo.getInstance(ASN1Sequence.getInstance(pkcs8Bytes));
      String oid = keyInfo.getPrivateKeyAlgorithm().getAlgorithm().getId();
      if (PKCSObjectIdentifiers.rsaEncryption.getId().equals(oid)) {
        return "RSA";
      } else if (X9ObjectIdentifiers.id_ecPublicKey.getId().equals(oid)) {
        return "EC";
      }
      throw new IllegalArgumentException("Unsupported key algorithm OID: " + oid);
    } catch (IllegalArgumentException e) {
      throw e;
    } catch (Exception e) {
      throw new IllegalArgumentException("Failed to detect key algorithm from PKCS8 encoding", e);
    }
  }

  /** Returns the crypto provider (cached after first lookup). BCFIPS when available, else default. */
  static java.security.Provider getCryptoProvider() {
    java.security.Provider provider = cachedProvider;
    if (provider != null) {
      return provider;
    }
    java.security.Provider bcfips = java.security.Security.getProvider(BCFIPS_PROVIDER);
    if (bcfips != null) {
      cachedProvider = bcfips;
      return bcfips;
    }
    provider = java.security.Security.getProviders()[0];
    cachedProvider = provider;
    return provider;
  }

  private static boolean isFipsProviderAvailable() {
    return getCryptoProvider().getName().equals(BCFIPS_PROVIDER);
  }
}
