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
import org.bouncycastle.asn1.pkcs.PrivateKeyInfo;
import org.bouncycastle.asn1.pkcs.RSAPrivateKey;
import org.bouncycastle.asn1.x509.AlgorithmIdentifier;
import org.bouncycastle.asn1.x9.X9ObjectIdentifiers;
import org.bouncycastle.util.io.pem.PemObject;
import org.bouncycastle.util.io.pem.PemReader;

public class FipsPemUtils {

  private static final String BCFIPS_PROVIDER = "BCFIPS";

  private FipsPemUtils() {}

  public static X509Certificate parseCertificate(String pem) throws CertificateException {
    if (pem == null || pem.isBlank()) {
      throw new CertificateException("PEM input is null or empty");
    }
    CertificateFactory factory = CertificateFactory.getInstance("X.509", getBcfipsProvider());
    byte[] derBytes = decodePem(pem, "CERTIFICATE");
    return (X509Certificate)
        factory.generateCertificate(new ByteArrayInputStream(derBytes));
  }

  public static PrivateKey parsePrivateKey(String pem)
      throws NoSuchAlgorithmException, InvalidKeySpecException, NoSuchProviderException,
          IOException {
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
          new AlgorithmIdentifier(org.bouncycastle.asn1.pkcs.PKCSObjectIdentifiers.rsaEncryption);
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
    KeyFactory keyFactory = KeyFactory.getInstance(algorithm, BCFIPS_PROVIDER);
    return keyFactory.generatePrivate(keySpec);
  }

  private static String detectKeyAlgorithm(byte[] pkcs8Bytes) {
    try {
      PrivateKeyInfo keyInfo = PrivateKeyInfo.getInstance(ASN1Sequence.getInstance(pkcs8Bytes));
      String oid = keyInfo.getPrivateKeyAlgorithm().getAlgorithm().getId();
      if ("1.2.840.113549.1.1.1".equals(oid)) {
        return "RSA";
      } else if ("1.2.840.10045.2.1".equals(oid)) {
        return "EC";
      }
      return "RSA";
    } catch (Exception e) {
      return "RSA";
    }
  }

  private static byte[] decodePem(String pem, String expectedType) {
    try (PemReader pemReader = new PemReader(new StringReader(pem))) {
      PemObject pemObject = pemReader.readPemObject();
      if (pemObject == null) {
        throw new IllegalArgumentException(
            "No PEM object found in input. Expected type: " + expectedType);
      }
      return pemObject.getContent();
    } catch (IOException e) {
      throw new IllegalArgumentException("Failed to read PEM input: " + e.getMessage(), e);
    }
  }

  private static java.security.Provider getBcfipsProvider() {
    java.security.Provider provider = java.security.Security.getProvider(BCFIPS_PROVIDER);
    if (provider == null) {
      throw new IllegalStateException(
          "BCFIPS provider not found. Ensure FipsSecurityProviderConfig.initialize() has been called.");
    }
    return provider;
  }
}
