// Package ethdss implements the Distributed Schnorr Signature protocol from the
////////////////////////////////////////////////////////////////////////////////
//       XXX: Do not use in production until this code has been audited.
////////////////////////////////////////////////////////////////////////////////
// paper "Provably Secure Distributed Schnorr Signatures and a (t, n)
// Threshold Scheme for Implicit Certificates".
// https://dl.acm.org/citation.cfm?id=678297
// To generate a distributed signature from a group of participants, the group
// must first generate one longterm distributed secret with the share/dkg
// package, and then one random secret to be used only once.
// Each participant then creates a DSS struct, that can issue partial signatures
// with `dss.PartialSignature()`. These partial signatures can be broadcasted to
// the whole group or to a trusted combiner. Once one has collected enough
// partial signatures, it is possible to compute the distributed signature with
// the `Signature` method.
//
// This is mostly copied from the sign/dss package, with minor adjustments for
// use with ethschnorr.
package ethdss

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/DCMMC/chainlink/core/services/signatures/ethschnorr"
	"github.com/DCMMC/chainlink/core/services/signatures/secp256k1"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

// Suite represents the functionalities needed by the dss package
type Suite interface {
	kyber.Group
	kyber.HashFactory
	kyber.Random
}

var secp256k1Suite = secp256k1.NewBlakeKeccackSecp256k1()
var secp256k1Group kyber.Group = secp256k1Suite

// DistKeyShare is an abstraction to allow one to use distributed key share
// from different schemes easily into this distributed threshold Schnorr
// signature framework.
type DistKeyShare interface {
	PriShare() *share.PriShare
	Commitments() []kyber.Point
}

// DSS holds the information used to issue partial signatures as well as to
// compute the distributed schnorr signature.
type DSS struct {
	// Keypair for this participant in the signing process (i.e., the one where
	// this struct is stored.) This is not the keypair for full signing key; that
	// would defeat the point.
	secret kyber.Scalar
	public kyber.Point
	// Index value of this participant in the signing process. The index is shared
	// across participants.
	index int
	// Public keys of potential participants in the signing process
	participants []kyber.Point
	// Number of participants needed to construct a signature
	T int
	// Shares of the distributed long-term signing keypair
	long DistKeyShare
	// Shares of the distributed ephemeral nonce keypair
	random DistKeyShare
	// Pedersen commitments to the coefficients of the polynomial implicitly used
	// to share the long-term signing public/private keypair.
	longPoly *share.PubPoly
	// Pedersen commitments to the coefficients of the polynomial implicitly used
	// to share the ephemeral nonce keypair.
	randomPoly *share.PubPoly
	// Message to be signed
	msg *big.Int
	// The partial signatures collected so far.
	partials []*share.PriShare
	// Indices for the participants who have provided their partial signatures to
	// this participant.
	partialsIdx map[int]bool
	// True iff the partial signature for this dss has been signed by its owner.
	signed bool
	// String which uniquely identifies this signature, shared by all
	// participants.
	sessionID []byte
}

// DSSArgs is the arguments to NewDSS, as a struct. See NewDSS for details.
type DSSArgs = struct {
	secret       kyber.Scalar
	participants []kyber.Point
	long         DistKeyShare
	random       DistKeyShare
	msg          *big.Int
	T            int
}

// PartialSig is partial representation of the final distributed signature. It
// must be sent to each of the other participants.
type PartialSig struct {
	Partial   *share.PriShare
	SessionID []byte
	Signature ethschnorr.Signature
}

// NewDSS returns a DSS struct out of the suite, the longterm secret of this
// node, the list of participants, the longterm and random distributed key
// (generated by the dkg package), the message to sign and finally the T
// threshold. It returns an error if the public key of the secret can't be found
// in the list of participants.
func NewDSS(args DSSArgs) (*DSS, error) {
	public := secp256k1Group.Point().Mul(args.secret, nil)
	var i int
	var found bool
	for j, p := range args.participants {
		if p.Equal(public) {
			found = true
			i = j
			break
		}
	}
	if !found {
		return nil, errors.New("dss: public key not found in list of participants")
	}
	return &DSS{
		secret:       args.secret,
		public:       public,
		index:        i,
		participants: args.participants,
		long:         args.long,
		longPoly: share.NewPubPoly(secp256k1Suite,
			secp256k1Group.Point().Base(), args.long.Commitments()),
		random: args.random,
		randomPoly: share.NewPubPoly(secp256k1Suite,
			secp256k1Group.Point().Base(), args.random.Commitments()),
		msg:         args.msg,
		T:           args.T,
		partialsIdx: make(map[int]bool),
		sessionID:   sessionID(secp256k1Suite, args.long, args.random),
	}, nil
}

// PartialSig generates the partial signature related to this DSS. This
// PartialSig can be broadcasted to every other participant or only to a
// trusted combiner as described in the paper.
// The signature format is compatible with EdDSA verification implementations.
//
// Corresponds to section 4.2, step 2 the Stinson 2001 paper.
func (d *DSS) PartialSig() (*PartialSig, error) {
	secretPartialLongTermKey := d.long.PriShare().V     // ɑᵢ, in the paper
	secretPartialCommitmentKey := d.random.PriShare().V // βᵢ, in the paper
	fullChallenge := d.hashSig()                        // h(m‖V), in the paper
	secretChallengeMultiple := secp256k1Suite.Scalar().Mul(
		fullChallenge, secretPartialLongTermKey) // ɑᵢh(m‖V)G, in the paper
	// Corresponds to ɣᵢG=βᵢG+ɑᵢh(m‖V)G in the paper, but NB, in its notation, we
	// use ɣᵢG=βᵢG-ɑᵢh(m‖V)G. (Subtract instead of add.)
	partialSignature := secp256k1Group.Scalar().Sub(
		secretPartialCommitmentKey, secretChallengeMultiple)
	ps := &PartialSig{
		Partial:   &share.PriShare{V: partialSignature, I: d.index},
		SessionID: d.sessionID,
	}
	var err error
	ps.Signature, err = ethschnorr.Sign(d.secret, ps.Hash()) // sign share
	if !d.signed {
		d.partialsIdx[d.index] = true
		d.partials = append(d.partials, ps.Partial)
		d.signed = true
	}
	return ps, err
}

// ProcessPartialSig takes a PartialSig from another participant and stores it
// for generating the distributed signature. It returns an error if the index is
// wrong, or the signature is invalid or if a partial signature has already been
// received by the same peer. To know whether the distributed signature can be
// computed after this call, one can use the `EnoughPartialSigs` method.
//
// Corresponds to section 4.3, step 3 of the paper
func (d *DSS) ProcessPartialSig(ps *PartialSig) error {
	var err error
	public, ok := findPub(d.participants, ps.Partial.I)
	if !ok {
		err = errors.New("dss: partial signature with invalid index")
	}
	// nothing secret here
	if err == nil && !bytes.Equal(ps.SessionID, d.sessionID) {
		err = errors.New("dss: session id do not match")
	}
	if err == nil {
		if vrr := ethschnorr.Verify(public, ps.Hash(), ps.Signature); vrr != nil {
			err = vrr
		}
	}
	if err == nil {
		if _, ok := d.partialsIdx[ps.Partial.I]; ok {
			err = errors.New("dss: partial signature already received from peer")
		}
	}
	if err != nil {
		return err
	}
	hash := d.hashSig() // h(m‖V), in the paper's notation
	idx := ps.Partial.I
	// βᵢG=sum(cₖi^kG), in the paper, defined as sᵢ in step 2 of section 2.4
	randShare := d.randomPoly.Eval(idx)
	// ɑᵢG=sum(bₖi^kG), defined as sᵢ in step 2 of section 2.4
	longShare := d.longPoly.Eval(idx)
	// h(m‖V)(Y+...) term from equation (3) of the paper. AKA h(m‖V)ɑᵢG
	challengeSummand := secp256k1Group.Point().Mul(hash, longShare.V)
	// RHS of equation (3), except we subtract the second term instead of adding.
	// AKA (βᵢ-ɑᵢh(m‖V))G, which should equal ɣᵢG, according to equation (3)
	maybePartialSigCommitment := secp256k1Group.Point().Sub(randShare.V,
		challengeSummand)
	// Check that equation (3) holds (ɣᵢ is represented as ps.Partial.V, here.)
	partialSigCommitment := secp256k1Group.Point().Mul(ps.Partial.V, nil)
	if !partialSigCommitment.Equal(maybePartialSigCommitment) {
		return errors.New("dss: partial signature not valid")
	}
	d.partialsIdx[ps.Partial.I] = true
	d.partials = append(d.partials, ps.Partial)
	return nil
}

// EnoughPartialSig returns true if there are enough partial signature to compute
// the distributed signature. It returns false otherwise. If there are enough
// partial signatures, one can issue the signature with `Signature()`.
func (d *DSS) EnoughPartialSig() bool {
	return len(d.partials) >= d.T
}

// Signature computes the distributed signature from the list of partial
// signatures received. It returns an error if there are not enough partial
// signatures.
//
// Corresponds to section 4.2, step 4 of Stinson, 2001 paper
func (d *DSS) Signature() (ethschnorr.Signature, error) {
	if !d.EnoughPartialSig() {
		return nil, errors.New("dkg: not enough partial signatures to sign")
	}
	// signature corresponds to σ in step 4 of section 4.2
	signature, err := share.RecoverSecret(secp256k1Suite, d.partials, d.T,
		len(d.participants))
	if err != nil {
		return nil, err
	}
	rv := ethschnorr.NewSignature()
	rv.Signature = secp256k1.ToInt(signature)
	// commitmentPublicKey corresponds to V in step 4 of section 4.2
	commitmentPublicKey := d.random.Commitments()[0]
	rv.CommitmentPublicAddress = secp256k1.EthereumAddress(commitmentPublicKey)
	return rv, nil
}

// hashSig returns, in the paper's notation, h(m‖V). It is the challenge hash
// for the signature. (Actually, the hash also includes the public key, but that
// has no effect on the correctness or robustness arguments from the paper.)
func (d *DSS) hashSig() kyber.Scalar {
	v := d.random.Commitments()[0] // Public-key commitment, in signature from d
	vAddress := secp256k1.EthereumAddress(v)
	publicKey := d.long.Commitments()[0]
	rv, err := ethschnorr.ChallengeHash(publicKey, vAddress, d.msg)
	if err != nil {
		panic(err)
	}
	return rv
}

// Verify takes a public key, a message and a signature and returns an error if
// the signature is invalid.
func Verify(public kyber.Point, msg *big.Int, sig ethschnorr.Signature) error {
	return ethschnorr.Verify(public, msg, sig)
}

// Hash returns the hash representation of this PartialSig to be used in a
// signature.
func (ps *PartialSig) Hash() *big.Int {
	h := secp256k1Suite.Hash()
	_, _ = h.Write(ps.Partial.Hash(secp256k1Suite))
	_, _ = h.Write(ps.SessionID)
	return (&big.Int{}).SetBytes(h.Sum(nil))
}

func findPub(list []kyber.Point, i int) (kyber.Point, bool) {
	if i >= len(list) {
		return nil, false
	}
	return list[i], true
}

func sessionID(s Suite, a, b DistKeyShare) []byte {
	h := s.Hash()
	for _, p := range a.Commitments() {
		_, _ = p.MarshalTo(h)
	}

	for _, p := range b.Commitments() {
		_, _ = p.MarshalTo(h)
	}

	return h.Sum(nil)
}
